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
        games = gameList !== null ? gameList.sort((a, b) => a.title > b.title ? 1 : -1) : [];
    };

    const render = () => {
        log.debug('[games] load game menu');
        listBox.innerHTML = games
            .map(game => `<div class="menu-item"><div><span>${game.title}</span></div><div class="menu-item__info">${game.system}</div></div>`)
            .join('');
    };

    const getTitleEl = (parent) => parent.firstChild.firstChild
    const getDescEl = (parent) => parent.children[1]

    const show = () => {
        render();
        gamesElList = listBox.querySelectorAll(`.menu-item`);
        menuItemChoice.style.display = "block";
        pickGame();
    };

    const bounds = (i = gameIndex) => (i % games.length + games.length) % games.length
    const clearPrev = () => {
        let prev = gamesElList[gameIndex]
        if (prev) {
            getTitleEl(prev).classList.remove('pick', 'text-move');
            getDescEl(prev).style.display = 'none'
        }
    }

    const pickGame = (index) => {
        clearPrev()
        gameIndex = bounds(index)

        const i = gamesElList[gameIndex];
        if (i) {
            const title = getTitleEl(i)
            setTimeout(() => {
                title.classList.add('pick')
                !menuInSelection && (getDescEl(i).style.display = 'block')
            }, 50)
            !menuInSelection && title.classList.add('text-move')
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
        const item = gamesElList[gameIndex]
        if (item) {
            getTitleEl(item).classList.add('text-move')
            getDescEl(item).style.display = 'block'
        }

        if (gamePickTimer === null) return;
        clearInterval(gamePickTimer);
        gamePickTimer = null;
    };

    const onMenuPressed = (newPosition) => {
        clearPrev(true)
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
        getCurrentGame: () => games[gameIndex].title
    }
})(document, event, log);
