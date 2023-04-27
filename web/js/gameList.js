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
    const MENU_SELECT_THRESHOLD_MS = 180;
    let menuTop = MENU_TOP_POSITION;
    let menuInSelection = false;
    const MENU_TRANSITION_DEFAULT = `top ${MENU_SELECT_THRESHOLD_MS}ms`;

    listBox.style.transition = MENU_TRANSITION_DEFAULT;

    let gamesElList;

    const setGames = (gameList) => {
        games = gameList !== null ? gameList.sort((a, b) => a > b ? 1 : -1) : [];
    };

    const render = () => {
        log.debug('[games] load game menu');
        listBox.innerHTML = games
            .map(game => `<div class="menu-item"><div><span>${game}</span></div></div>`)
            .join('');
    };

    const show = () => {
        render();
        gamesElList = listBox.querySelectorAll(`.menu-item span`);
        menuItemChoice.style.display = "block";
        pickGame();
    };

    const bounds = (i = gameIndex) => (i % games.length + games.length) % games.length
    const clearPrev = () => {
        let prev = gamesElList[gameIndex]
        if (prev) {
            prev.classList.remove('pick', 'text-move');
        }
    }

    const pickGame = (index) => {
        clearPrev()
        gameIndex = bounds(index)

        const i = gamesElList[gameIndex];
        if (i) {
            setTimeout(() => i.classList.add('pick'), 50)
            !menuInSelection && i.classList.add('text-move')
        }

        // transition menu box
        menuTop = MENU_TOP_POSITION - gameIndex * 36;
        listBox.style.top = `${menuTop}px`;
    };

    const startGamePickerTimer = (upDirection) => {
        menuInSelection = true
        if (gamePickTimer !== null) return;
        const shift = upDirection ? -1 : 1;
        pickGame(gameIndex + shift);

        // velocity?
        // keep rolling the game list if the button is pressed
        gamePickTimer = setInterval(() => {
            pickGame(gameIndex + shift, true);
        }, MENU_SELECT_THRESHOLD_MS);
    };

    const stopGamePickerTimer = () => {
        menuInSelection = false
        gamesElList[gameIndex] && gamesElList[gameIndex].classList.add('text-move')

        if (gamePickTimer === null) return;
        clearInterval(gamePickTimer);
        gamePickTimer = null;
    };

    const onMenuPressed = (newPosition) => {
        clearPrev()
        listBox.style.transition = '';
        listBox.style.top = `${menuTop - newPosition}px`;
    };

    const onMenuReleased = (position) => {
        listBox.style.transition = MENU_TRANSITION_DEFAULT
        menuTop -= position;
        pickGame(Math.round((menuTop - MENU_TOP_POSITION) / -36));
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
