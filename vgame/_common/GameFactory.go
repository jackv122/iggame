package com

var GetGameInterface func(gameId GameId, gameServer *GameServer) GameInterface

var GameFactory func(gameId GameId, gameServer *GameServer) GameInterface
