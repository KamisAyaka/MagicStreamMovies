import axiosClient from "../../api/axiosConfig";
import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { Form, Button, Container } from "react-bootstrap";

const Register = () => {
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [favouriteGenres, setFavouriteGenres] = useState([]);
  const [genres, setGenres] = useState([]);

  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleGenreChange = (e) => {
    const option = Array.from(e.target.selectedOptions);
    setFavouriteGenres(
      option.map((opt) => ({
        genre_id: Number(opt.value),
        genre_name: opt.label,
      }))
    );
  };
  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    const defaultRole = "USER";
    console.log(defaultRole);

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }
    setLoading(true);
    try {
      const payload = {
        first_name: firstName,
        last_name: lastName,
        email,
        password,
        role: defaultRole,
        favourite_genres: favouriteGenres,
      };
      const response = await axiosClient.post("/register", payload);
      if (response.data.error) {
        setError(response.data.error);
        return;
      }
      navigate("/login", { replace: true });
    } catch (error) {
      console.error("Error registering user:", error);
      setError(error.response.data.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const fetchGenres = async () => {
      try {
        const response = await axiosClient.get("/genres");
        setGenres(response.data);
      } catch (error) {
        console.error("Error fetching genres:", error);
      }
    };
    fetchGenres();
  }, []);

  return (
    <Container className="login-container d-flex align-items-center justify-content-center min-vh-100">
      <div
        className="w-100 max-w-md p-4 bg-white rounded shadow"
        style={{ maxWidth: "400px", width: "100%" }}
      >
        <div className="text-center mb-4">
          <h2 className="fw-bold mb-2">Create Account</h2>
          <p className="text-muted">
            Join MagicStreamMovies to discover your favorite movies and TV shows
          </p>
          {error && <div className="alert alert-danger">{error}</div>}
        </div>

        <Form onSubmit={handleSubmit}>
          <Form.Group className="mb-3">
            <Form.Label>First Name</Form.Label>
            <Form.Control
              type="text"
              placeholder="Enter first name"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Last Name</Form.Label>
            <Form.Control
              type="text"
              placeholder="Enter last name"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Email</Form.Label>
            <Form.Control
              type="email"
              placeholder="Enter email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Password</Form.Label>
            <Form.Control
              type="password"
              placeholder="Enter password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Confirm Password</Form.Label>
            <Form.Control
              type="password"
              placeholder="Confirm password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              isInvalid={password !== confirmPassword && confirmPassword !== ""}
            />
            <Form.Control.Feedback type="invalid">
              Passwords do not match
            </Form.Control.Feedback>
          </Form.Group>
          <Form.Group>
            <Form.Select
              multiple
              value={favouriteGenres.map((g) => String(g.genre_id))}
              onChange={handleGenreChange}
            >
              {genres.map((genre) => (
                <option
                  key={genre.genre_id}
                  value={genre.genre_id}
                  label={genre.genre_name}
                >
                  {genre.genre_name}
                </option>
              ))}
            </Form.Select>
            <Form.Text className="text-muted">
              Hold Ctrl or Command to select multiple genres
            </Form.Text>
          </Form.Group>
          <Button
            variant="primary"
            type="submit"
            className="w-100 mb-2"
            disabled={loading}
            style={{ fontWeight: 600, letterSpacing: 1 }}
          >
            {loading ? "Registering..." : "Register"}
          </Button>
          {error && <p className="text-danger">{error}</p>}
          <p className="text-muted">
            Already have an account? <Link to="/login">Login</Link>
          </p>
        </Form>
      </div>
    </Container>
  );
};
export default Register;
