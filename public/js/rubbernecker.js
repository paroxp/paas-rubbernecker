var App = {
    config: {
        updateCheck: null,
        updatePending: null,
        countdowns: []
    },

    checkForUpdates: function () {
        if (App.config.updatePending !== null) {
            return;
        }

        $.ajax({
            url: "/",
            type: "GET",
            beforeSend: function (xhr) {
                xhr.setRequestHeader('x-rubbernecker-state', currentState);
            },
            success: function (data, textStatus, xhr) {
                if (data.state === 'dirty') {
                    $('.update').slideDown();
                    App.config.updatePending = setTimeout(App.refresh, 30 * 1000); // Refresh page in about 30 seconds.
                    App.setupCountdown('state', 30, '.update p span');
                }
            },
            error: function (jqXHR, textStatus, errorThrown) {
                console.error('Failed to obtain the update response.');
                console.debug(jqXHR, textStatus, errorThrown);
            }
        });
    },

    disableUpdates: function () {
        if (App.config.updateCheck !== null) {
            clearInterval(App.config.updateCheck);
            App.config.updateCheck = null;
        }

        if (App.config.updatePending !== null) {
            clearTimeout(App.config.updatePending);
            App.config.updatePending = null;
        }

        $('.board-updates [data-switch="on"]').removeClass('active');
        $('.board-updates [data-switch="off"]').addClass('active');

        $('.update').slideUp();
    },

    enableUpdates: function (seconds) {
        if (App.config.updateCheck === null) {
            App.config.updateCheck = setInterval(App.checkForUpdates, seconds * 1000);
        }
    },

    gracefulIn: function ($elements) {
        $elements.each(function () {
            var $element = $(this);

            if (!$element.is(':hidden')) {
                return;
            }

            $element.css('opacity', 0);
            $element.slideDown();

            setTimeout(function () {
                $element.animate({
                    opacity: 1
                });
            }, 500);
        });
    },

    gracefulOut: function ($elements) {
        $elements.each(function () {
            var $element = $(this);

            if ($element.is(':hidden')) {
                return;
            }

            $element.css('opacity', 1);
            $element.animate({
                opacity: 0
            });

            setTimeout(function () {
                $element.slideUp();
            }, 500);
        });
    },

    refresh: function () {
        location.reload();
    },

    setupCountdown: function (name, seconds, element) {
        $(element).attr('data-seconds', seconds).text(seconds);

        clearInterval(App.config.countdowns[name]);

        App.config.countdowns[name] = setInterval(function () {
            var left = $(element).attr('data-seconds') - 1;

            if (left < 0) {
                return;
            }

            $(element).attr('data-seconds', left).text(left);
        }, 1000);
    },

    toggleMenu: function (e) {
        e.preventDefault();

        $('div.options').toggleClass('active');

        return false;
    },

    toggleCards: function (e) {
        App.toggleSwitch(e);

        var target = $(this).attr('data-target'),
            toHide = $(this).attr('data-hide');

        // Show all the stories.
        App.gracefulIn($(target));

        // Hide other elements if possible.
        if (toHide) {
            App.gracefulOut($(target).filter(toHide));
        }
    },

    toggleSwitch: function (e) {
        e.preventDefault();

        var $switches = $(this).parent().find('a[data-switch]');

        if ($(this).hasClass('active')) {
            return;
        }

        // Toggle active class.
        $switches.filter('.active').removeClass('active');
        $(this).addClass('active');

        return false;
    }
};

$(document)
    .ready(function () {
        $('body')
            .on('click', '[data-switch]', App.toggleSwitch)
            .on('click', '.team-switch a', App.toggleCards)
            .on('click', 'div.options .handler', App.toggleMenu)
            .on('click', '.update [data-trigger="cancel"], .board-updates [data-switch="off"]', App.disableUpdates)
            .on('click', '.board-updates [data-switch="on"]', setupUpdates)
            .on('click', '.update [data-trigger="refresh"]', App.refresh);

        setupUpdates();

        //////////

        function setupUpdates() {
            App.enableUpdates(30); // Enable updates every 30 seconds.
        }
    });