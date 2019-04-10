// miscs
DEBUG = true;

// list game
GAME_LIST = [
    {
        name: "Contra",
        nes: "Contra.nes",
        art: "Contra (USA).png"
    },

    {
        name: "Kirby's Adventure",
        nes: "Kirby's Adventure.nes",
        art: "Kirby's Adventure (Canada).png"
    },

    {
        name: "Mega Man 2",
        nes: "Mega Man 2.nes",
        art: "Mega Man 2 (USA).png"
    },

    {
        name: "Metal Gear",
        nes: "Metal Gear.nes",
        art: "Metal Gear (USA).png"
    },

    {
        name: "Mortal Kombat 4",
        nes: "Mortal Kombat 4.nes",
        art: "mortal-kombat-x-box-art-revealed-as-pre-orders-ope_5pxd.jpg"
    },

    {
        name: "Super Mario Bros 2",
        nes: "Super Mario Bros 2.nes",
        art: "Super Mario Bros. 2 (USA).png"
    },

    {
        name: "Super Mario Bros 3",
        nes: "Super Mario Bros 3.nes",
        art: "Super Mario Bros. 3 (USA).png"
    },

    {
        name: "Super Mario Bros",
        nes: "Super Mario Bros.nes",
        art: "Super Mario Bros. (World).png"
    },

    {
        name: "TMNT 3",
        nes: "Teenage Mutant Ninja Turtles 3.nes",
        art: "Teenage Mutant Ninja Turtles III - The Manhattan Project (USA).png"
    },

    {
        name: "Zelda",
        nes: "zelda.rom",
        art: "Zelda II - The Adventure of Link (Europe) (Rev A).png"
    }
];



// Keyboard stuffs
KEY_MAP = {
    37: "left",
    38: "up",
    39: "right",
    40: "down",

    90: "a", // z
    88: "b", // x
    67: "start", // c
    86: "select", // v

    81: "quit", // q
    83: "save", // s
    76: "load" // l
}

/*
      const (
        ButtonA = iota
        ButtonB
        ButtonSelect
        ButtonStart
        ButtonUp
        ButtonDown
        ButtonLeft
        ButtonRight
      )
      */
KEY_BIT = ["a", "b", "select", "start", "up", "down", "left", "right", "save", "load"];


INPUT_FPS = 100;
INPUT_STATE_PACKET = 5;
