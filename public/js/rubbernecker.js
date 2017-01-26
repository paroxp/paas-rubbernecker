var App = {
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

    toggleMenu: function (e) {
        e.preventDefault();

        $('div.options').toggleClass('active');

        return false;
    },

    toggleTeam: function (e) {
        e.preventDefault();

        var $switches = $('.team-switch a'),
            target = $(this).attr('data-target'),
            toHide = $(this).attr('data-hide');

        if ($(this).hasClass('active')) {
            return;
        }

        // Toggle active class.
        $switches.filter('.active').removeClass('active');
        $(this).addClass('active');

        // Show all the stories.
        App.gracefulIn($(target));

        // Hide other elements if possible.
        if (toHide) {
            App.gracefulOut($(target).filter(toHide));
        }

        return false;
    }
};

$(document)
    .ready(function () {
        $('body')
            .on('click', '.team-switch a', App.toggleTeam)
            .on('click', 'div.options .handler', App.toggleMenu);
    });