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
    const listBox = document.getElementById('menu-container');
    const menuItemChoice = document.getElementById('menu-item-choice');

    const MENU_TOP_POSITION = 102;
    let menuTop = MENU_TOP_POSITION;

    const setGames = (gameList) => {
        games = gameList.sort((a, b) => a > b ? 1 : -1);
    };

    const render = () => {
        log.debug('[games] load game menu');

        listBox.innerHTML = games
            .map(game => `<div class="menu-item"><div><span>${game}</span></div></div>`)
            .join('');
    };

    const show = () => {
        render();
        menuItemChoice.style.display = "block";
        pickGame();
    };

    const pickGame = (index, hold) => {
        let idx = undefined !== index ? index : gameIndex;

        // check boundaries
        // cycle
        if (idx < 0) idx = games.length - 1;
        if (idx >= games.length) idx = 0;

        // transition menu box
        listBox.style['transition'] = 'top 0.2s';
        menuTop = MENU_TOP_POSITION - idx * 36;
        listBox.style['top'] = `${menuTop}px`;

        const cl = hold ? 'pick-over' : 'pick'
        let pick = listBox.querySelectorAll('.menu-item .pick, .menu-item .pick-over')[0];

        setTimeout(() => {        // overflow marquee
            if (pick) {
                pick.classList.remove('pick');
                pick.classList.remove('pick-over')
            }
            listBox.querySelectorAll(`.menu-item span`)[idx].classList.add(cl);
        }, 50)

            gameIndex = idx;
    };

    const startGamePickerTimer = (upDirection) => {
        if (gamePickTimer !== null) return;
        const shift = upDirection ? -1 : 1;
        pickGame(gameIndex + shift);

        // velocity?
        // keep rolling the game list if the button is pressed
        gamePickTimer = setInterval(() => {
            pickGame(gameIndex + shift, true);
        }, 150);
    };

    const stopGamePickerTimer = () => {
        if (gamePickTimer === null) return;
        clearInterval(gamePickTimer);
        gamePickTimer = null;
        let pick = listBox.querySelectorAll('.menu-item .pick-over')[0];

        pick.classList.remove('pick-over');
        pick.classList.add('pick');

    };

    const onMenuPressed = (newPosition) => {
        listBox.style['transition'] = '';
        listBox.style['top'] = `${menuTop - newPosition}px`;
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
        set: setGames,
        getCurrentGame: () => games[gameIndex]
    }
})(document, event, log);
