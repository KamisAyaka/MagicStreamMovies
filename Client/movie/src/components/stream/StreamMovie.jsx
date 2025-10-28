import { useParams } from "react-router-dom";
import { useState, useEffect } from "react";
import ReactPlayer from "react-player";
import "./StreamMovie.css";

const StreamMovie = () => {
  let params = useParams();
  let key = params.yt_id;
  const [error, setError] = useState(null);
  const [isReady, setIsReady] = useState(false);

  // 组件挂载时重置状态
  useEffect(() => {
    setError(null);
    setIsReady(false);
  }, [key]);

  const handleError = (e) => {
    console.error("视频播放错误:", e);
    setError("视频加载失败，请检查网络连接或稍后重试");
  };

  const handleReady = () => {
    setIsReady(true);
    setError(null);
  };

  return (
    <div className="react-player-container">
      {key != null ? (
        <>
          {!isReady && !error && (
            <div className="video-loading">
              <p>正在加载视频...</p>
            </div>
          )}
          {error && (
            <div className="video-error">
              <p>
                <i className="fa fa-exclamation-circle fa-3x"></i>
              </p>
              <p>{error}</p>
            </div>
          )}
          <ReactPlayer
            playing={true}
            controls={true}
            width="100%"
            height="100%"
            url={`https://www.youtube.com/watch?v=${key}`}
            onError={handleError}
            onReady={handleReady}
            config={{
              youtube: {
                playerVars: {
                  autoplay: 1,
                  modestbranding: 1,
                },
              },
            }}
          />
        </>
      ) : (
        <div className="no-video">
          <p>
            <i className="fa fa-exclamation-triangle fa-3x"></i>
          </p>
          <p>未找到视频</p>
        </div>
      )}
    </div>
  );
};

export default StreamMovie;
