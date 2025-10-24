import React, { useState, useEffect, useRef } from 'react';
import './App.css';
import Board from './Board';
import Leaderboard from './Leaderboard';

const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

function App() {
  const [username, setUsername] = useState('');
  const [gameState, setGameState] = useState(null);
  const [playerNum, setPlayerNum] = useState(null);
  const [opponent, setOpponent] = useState('');
  const [message, setMessage] = useState('');
  const [connected, setConnected] = useState(false);
  const [showLeaderboard, setShowLeaderboard] = useState(false);
  const ws = useRef(null);

  useEffect(() => {
    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, []);

  const connectWebSocket = () => {
    ws.current = new WebSocket(WS_URL);

    ws.current.onopen = () => {
      setConnected(true);
      console.log('WebSocket connected');
      // Send join message immediately after connection opens
      if (username.trim()) {
        ws.current.send(JSON.stringify({ type: 'join', username: username }));
        console.log('Join message sent:', username);
      }
    };

    ws.current.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      handleMessage(msg);
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      setMessage('Connection error. Please refresh.');
    };

    ws.current.onclose = () => {
      setConnected(false);
      console.log('WebSocket disconnected');
    };
  };

  const handleMessage = (msg) => {
    console.log('Received:', msg);

    switch (msg.type) {
      case 'waiting':
        setMessage('Waiting for opponent... (Bot will join in 10s if no player found)');
        break;

      case 'game_start':
        setPlayerNum(msg.data.playerNum);
        setOpponent(msg.data.opponent);
        setGameState(msg.data.gameState);
        setMessage(
          msg.data.opponentIsBot
            ? `Game started! Playing against BOT. You are Player ${msg.data.playerNum}.`
            : `Game started! Playing against ${msg.data.opponent}. You are Player ${msg.data.playerNum}.`
        );
        break;

      case 'game_update':
        setGameState(msg.data.gameState);
        if (msg.data.gameState.status === 'finished') {
          const winner = msg.data.gameState.winner;
          const winnerName = msg.data.gameState.winnerName || '';
          if (winner === 0) {
            setMessage("Game over! It's a draw!");
          } else if (winnerName) {
            // Show actual winner's name
            if (winner === playerNum) {
              setMessage(`Game over! ${winnerName} (You) won! 🎉`);
            } else {
              setMessage(`Game over! ${winnerName} won!`);
            }
          } else {
            // Fallback if winnerName not provided
            if (winner === playerNum) {
              setMessage('Game over! You won! 🎉');
            } else {
              setMessage(`Game over! ${opponent} won!`);
            }
          }
        } else {
          const isYourTurn = msg.data.gameState.currentTurn === playerNum;
          const opponentName = opponent || 'Opponent';
          setMessage(isYourTurn ? 'Your turn!' : `${opponentName}'s turn...`);
        }
        break;

      case 'reconnected':
        setPlayerNum(msg.data.playerNum);
        setGameState(msg.data.gameState);
        setMessage('Reconnected to game!');
        break;

      case 'error':
        setMessage(msg.data.message || 'An error occurred');
        break;

      default:
        console.log('Unknown message type:', msg.type);
    }
  };

  const handleJoin = () => {
    if (!username.trim()) {
      setMessage('Please enter a username');
      return;
    }

    setMessage('Connecting...');
    connectWebSocket();
  };

  const handleMove = (col) => {
    if (!gameState || gameState.status !== 'playing') {
      return;
    }

    if (gameState.currentTurn !== playerNum) {
      setMessage("Not your turn!");
      return;
    }

    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({ type: 'move', column: col }));
    }
  };

  const handleNewGame = () => {
    if (ws.current) {
      ws.current.close();
    }
    setGameState(null);
    setPlayerNum(null);
    setOpponent('');
    setMessage('');
    setConnected(false);
    setUsername('');
  };

  if (showLeaderboard) {
    return (
      <div className="App">
        <Leaderboard apiUrl={API_URL} onBack={() => setShowLeaderboard(false)} />
      </div>
    );
  }

  if (!connected || !gameState) {
    return (
      <div className="App">
        <div className="container">
          <h1>🎮 4 in a Row</h1>
          <div className="join-form">
            <input
              type="text"
              placeholder="Enter your username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleJoin()}
              disabled={connected}
            />
            <button onClick={handleJoin} disabled={connected || !username.trim()}>
              {connected ? 'Connecting...' : 'Join Game'}
            </button>
            <button onClick={() => setShowLeaderboard(true)} className="secondary">
              View Leaderboard
            </button>
          </div>
          {message && <div className="message">{message}</div>}
        </div>
      </div>
    );
  }

  return (
    <div className="App">
      <div className="container">
        <h1>🎮 4 in a Row</h1>
        <div className="game-info">
          <div>
            <strong>You:</strong> {username} (Player {playerNum})
          </div>
          <div>
            <strong>Opponent:</strong> {opponent}
          </div>
        </div>
        <div className="message">{message}</div>
        <Board
          board={gameState.board}
          onColumnClick={handleMove}
          currentTurn={gameState.currentTurn}
          playerNum={playerNum}
          gameStatus={gameState.status}
        />
        <div className="controls">
          <button onClick={handleNewGame}>New Game</button>
          <button onClick={() => setShowLeaderboard(true)} className="secondary">
            Leaderboard
          </button>
        </div>
      </div>
    </div>
  );
}

export default App;
