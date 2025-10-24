import React from 'react';
import './Board.css';

function Board({ board, onColumnClick, currentTurn, playerNum, gameStatus }) {
  const canPlay = gameStatus === 'playing' && currentTurn === playerNum;

  const handleColumnClick = (col) => {
    if (canPlay) {
      onColumnClick(col);
    }
  };

  const getCellClass = (value) => {
    if (value === 0) return 'cell empty';
    if (value === 1) return 'cell player1';
    if (value === 2) return 'cell player2';
    return 'cell';
  };

  return (
    <div className="board">
      {board && board.map((row, rowIndex) => (
        <div key={rowIndex} className="row">
          {row.map((cell, colIndex) => (
            <div
              key={colIndex}
              className={getCellClass(cell)}
              onClick={() => handleColumnClick(colIndex)}
              style={{ cursor: canPlay ? 'pointer' : 'default' }}
            >
              {cell !== 0 && <div className="disc"></div>}
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

export default Board;
