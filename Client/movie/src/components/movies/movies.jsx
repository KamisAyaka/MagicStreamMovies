import Movie from "../movie/movie";

const Movies = ({ movies, updateMovieReview, message }) => {
  return (
    <div className="container mt-4">
      <div className="row">
        {movies && movies.length > 0 ? (
          movies.map((movie) => (
            <Movie
              key={movie.imdb_id}
              updateMovieReview={updateMovieReview}
              movie={movie}
            />
          ))
        ) : (
          <h2>{message}</h2>
        )}
      </div>
    </div>
  );
};

export default Movies;
