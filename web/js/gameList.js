/**
 * Game list module.
 * @version 1
 */
const gameList = (() => {
    // state
    let games = [];
    let gameIndex = 1;
    let gamePickTimer = null;

    // UI
    const listBox = $('#menu-container');
    const menuItemChoice = $('#menu-item-choice');

    const MENU_TOP_POSITION = 102;
    let menuTop = MENU_TOP_POSITION;

    const setGames = (gameList) => {
        games = gameList.sort((a, b) => a > b ? 1 : -1);
    };

    const render = () => {
        log.debug('[games] load game menu');

        listBox.html(games
            .map(game => `<div class="menu-item unselectable" unselectable="on"><div><span>${game}</span></div></div>`)
            .join('')
        );
    };

    const show = () => {
        render();
        menuItemChoice.show();
        pickGame();
    };

    const hide = () => {
        menuItemChoice.hide();
    };

    const pickGame = (index) => {
        let idx = undefined !== index ? index : gameIndex;

        // check boundaries
        // cycle
        if (idx < 0) idx = games.length - 1;
        if (idx >= games.length) idx = 0;

        // transition menu box
        listBox.css('transition', 'top 0.2s');
        listBox.css('-moz-transition', 'top 0.2s');
        listBox.css('-webkit-transition', 'top 0.2s');

        menuTop = MENU_TOP_POSITION - idx * 36;
        listBox.css('top', `${menuTop}px`);

        // overflow marquee
        $('.menu-item .pick').removeClass('pick');
        $(`.menu-item:eq(${idx}) span`).addClass('pick');

        gameIndex = idx;
    };

    const startGamePickerTimer = (upDirection) => {
        if (gamePickTimer !== null) return;

        log.debug('[games] start game picker timer');
        const shift = upDirection ? -1 : 1;
        pickGame(gameIndex + shift);

        // velocity?
        // keep rolling the game list if the button is pressed
        gamePickTimer = setInterval(() => {
            pickGame(gameIndex + shift);
        }, 200);
    };

    const stopGamePickerTimer = () => {
        if (gamePickTimer === null) return;

        log.debug('[games] stop game picker timer');
        clearInterval(gamePickTimer);
        gamePickTimer = null;
    };

    const onMenuPressed = (newPosition) => {
        listBox.css('transition', '');
        listBox.css('-moz-transition', '');
        listBox.css('-webkit-transition', '');
        listBox.css('top', `${menuTop - newPosition}px`);
    };

    const onMenuReleased = (position) => {
        menuTop -= position;
        const index = Math.round((menuTop - MENU_TOP_POSITION) / -36);
        pickGame(index);
    };

    event.sub(MENU_PRESSED, onMenuPressed);
    event.sub(MENU_RELEASED, onMenuReleased);

    return {
        startGamePickerTimer: startGamePickerTimer,
        stopGamePickerTimer: stopGamePickerTimer,
        pickGame: pickGame,
        show: show,
        hide: hide,
        set: setGames,
        getCurrentGame: () => games[gameIndex]
    }
})($, event, log);
