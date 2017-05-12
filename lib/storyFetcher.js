var request = require('request');
var async = require('async');
var personFetcher = require('./personFetcher');
var findBusinessDaysInRange = require('find-business-days-in-range').calc;
var crypto = require('crypto');
var stickers = require('./stickers.js').stickers;

module.exports = storyFetcher;

function storyFetcher() {};

var internals = {};
const TRANSITION_ERROR_CODE = -99;

internals.getLead = function(story) {
    var lead;
    story.labels.forEach(function(label) {
        var re = new RegExp('tech lead: ([a-z]+)', 'gi');
        var matches = re.exec(label.name);
        if(matches) {
            lead = matches[1].charAt(0).toUpperCase() + matches[1].slice(1).toLowerCase();
        }
    });
    return lead;
}

internals.obtainLabels = function (story) {
    var data = [];

    story.labels.forEach(function (label) {
        if (typeof stickers[label.name] !== "undefined") {
            data.push(stickers[label.name]);
        }
    });

    return data;
}

internals.getStoryViewModel = function (membershipInfo, storyDetail, transitions) {
    var viewModels = storyDetail.map(function (story) {
        var workers = story.owner_ids.map(function (worker_id) {
            return personFetcher.mapPersonFromId(worker_id, membershipInfo);
        });
        var daysInProgress = internals.calculateDaysInProgress(story, transitions);
        return {
            id: story.id,
            name: story.name,
            workers: workers,
            daysInProgress: daysInProgress,
            lead: internals.getLead(story),
            status: story.current_state,
            stickers: internals.obtainLabels(story)
        }
    });

    return viewModels;
}

internals.getStoriesByStatus = function(res, callback, status) {
    // Get the list of stories
    var options = {
        url: 'https://www.pivotaltracker.com/services/v5/projects/' + res.app.get('pivotalProjectId') + '/stories?date_format=millis&with_state=' + status,
        headers: {
            'X-TrackerToken': res.app.get('pivotalApiKey')
        }
    };

    request(options, function getStories(error, response, body) {
        if (!error && response.statusCode == 200) {
            var stories = res.app.get('stories');
            stories[status] = JSON.parse(body);
            res.app.set('stories', stories);

            callback(null, stories[status]);
        } else {
            callback("Couldn't get stories thanks to this crap: " + response.statusCode, null);
        }
    });
}

internals.getStoryTransitions = function(res, cb) {
    var appStories = res.app.get('stories');

    var calls = [];
    var stories = appStories.started
        .concat(appStories.finished)
        .concat(appStories.delivered)
        .concat(appStories.rejected);

    stories.forEach(function (story) {
        calls.push(function (callback) {
            var options = {
                url: 'https://www.pivotaltracker.com/services/v5/projects/' + res.app.get('pivotalProjectId') + '/stories/' + story.id + '/transitions',
                headers: {
                    'X-TrackerToken': res.app.get('pivotalApiKey')
                }
            };

            request(options, function getStories(error, response, body) {
                if (!error && response.statusCode == 200) {
                    callback(null, JSON.parse(body));
                } else {
                    callback("Could not load the transitions for the story: " + story.id, transitions);
                }
            });
        });
    });

    async.parallel(calls, function (err, results) {
        var transitions = [];

        results.forEach(function (result) {
            transitions = transitions.concat(result);
        });

        cb(transitions);
    });
}

internals.calculateDaysInProgress = function (story, transitions) {
    // Gather relevant transitions
    var storyTransitions = transitions.filter(function (transition) {
        return transition.story_id === story.id && transition.state === story.current_state;
    });

    // assume an error until we find a valid value
    // var diffDays = TRANSITION_ERROR_CODE;
    var diffDays,
        mostRecentTransitionDate;

    if (storyTransitions.length > 0) {
        mostRecentTransitionDate = new Date(Math.max.apply(null, storyTransitions.map(function (transition) {
            return new Date(transition.occurred_at);
        })));
    } else {
        mostRecentTransitionDate = new Date(story.created_at);
    }

    diffDays = findBusinessDaysInRange(mostRecentTransitionDate, new Date()).length;

    if (diffDays < 0) {
        diffDays = 0;
    }

    return diffDays;
}

function tokenise(play, finished, delivered, rejected) {
    var collection = [];

    if (play.length > 0) {
        collection.push('(' + stringifyStories(play) + ')');
    }
    if (finished.length > 0) {
        collection.push('(' + stringifyStories(finished) + ')');
    }
    if (delivered.length > 0) {
        collection.push('(' + stringifyStories(delivered) + ')');
    }
    if (rejected.length > 0) {
        collection.push('(' + stringifyStories(rejected) + ')');
    }

    return crypto.createHmac('sha512', collection.join('.')).digest('hex');

    //////////

    function stringifyStories(stories) {
        var collection = [];

        for (var i = 0; i < stories.length; i++) {
            var story = stories[i];
            var coll = [];
            coll.push(story.id);
            coll.push(story.team != undefined ? story.team : 'neutral');
            coll.push(story.status);
            coll.push(story.workers != undefined ? '[' + story.workers.join(':') + ']' : 'null');
            coll.push('[' + getLabels(story) + ']');

            collection.push(coll.join(':'));
        }

        return collection.join(';');
    }

    function getLabels(story) {
        if (story.labels === undefined) {
            return 'empty';
        }

        var collection = [];

        for (var i = 0; i < story.labels.length; i++) {
            var label = story.labels[i];
            collection.push(label.name);
        }

        return collection.join(':');
    }
}

storyFetcher.getStorySummary = function (req, res) {
    res.app.set('stories', res.app.get('stories') || {});

    async.parallel(
        {
            members: function (callback) {
                personFetcher.getMembers(res, callback);
            },
            started: function (callback) {
                internals.getStoriesByStatus(res, callback, "started");
            },
            finished: function (callback) {
                internals.getStoriesByStatus(res, callback, "finished");
            },
            delivered: function (callback) {
                internals.getStoriesByStatus(res, callback, "delivered");
            },
            rejected: function (callback) {
                internals.getStoriesByStatus(res, callback, "rejected");
            }
        },
        // Combine the results of the things above
        function (err, results) {
            if (err) {
                res.render('damn', {
                    message: '┬──┬◡ﾉ(° -°ﾉ)',
                    status: err,
                    reason: "(╯°□°）╯︵ ┻━┻"
                });
            } else {
                async.parallel([
                    function (callback) {
                        internals.getStoryTransitions(res, callback);
                    }
                ], function (transitions) {
                    var state = req.get('x-rubbernecker-state');

                    var startedStories = internals.getStoryViewModel(results.members, results.started, transitions);
                    var finishedStories = internals.getStoryViewModel(results.members, results.finished, transitions);
                    var deliveredStories = internals.getStoryViewModel(results.members, results.delivered, transitions);
                    var rejectedStories = internals.getStoryViewModel(results.members, results.rejected, transitions);
                    var reviewSlotsLimit = res.app.get('reviewSlotsLimit');
                    var approveSlotsLimit = res.app.get('signOffSlotsLimit');
                    var reviewSlotsFull = reviewSlotsLimit < finishedStories.length;
                    var approveSlotsFull = approveSlotsLimit < deliveredStories.length;
                    var currentState = tokenise(startedStories, finishedStories, deliveredStories, rejectedStories);

                    if (state != undefined) {
                        res.setHeader('Content-Type', 'application/json');
                        res.send(JSON.stringify({
                            state: state === currentState ? 'clean' : 'dirty'
                        }));

                        return;
                    }

                    res.render('index', {
                        projectId: res.app.get('pivotalProjectId'),
                        story: startedStories,
                        finishedStory: finishedStories,
                        deliveredStory: deliveredStories,
                        rejectedStory: rejectedStories,
                        reviewSlotsLimit: reviewSlotsLimit,
                        approveSlotsLimit: approveSlotsLimit,
                        reviewSlotsFull: reviewSlotsFull,
                        approveSlotsFull: approveSlotsFull,
                        currentState: currentState
                    });
                });
            }
        });
}
