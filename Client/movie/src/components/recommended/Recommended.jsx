import useAxiosPrivate from "../../hooks/useAxiosPrivate";
import { useState, useEffect } from "react";
import Movies from "../movies/movies";

const Recommended = () => {
  const [movies, setMovies] = useState([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState();
  const axiosPrivate = useAxiosPrivate();

  useEffect(() => {
    const fetchRecommendedMovies = async () => {
      setLoading(true);
      setMessage("");
      try {
        const response = await axiosPrivate.get("/recommendedmovies");
        setMovies(response.data);
      } catch (error) {
        console.error("Error fetching recommended movies:", error);
        setMessage(error.response.data.message);
      } finally {
        setLoading(false);
      }
    };
    fetchRecommendedMovies();
  }, []);
  return (
    <>
      {loading ? (
        <h2>Loading...</h2>
      ) : (
        <Movies movies={movies} message={message} />
      )}
    </>
  );
};

export default Recommended;
