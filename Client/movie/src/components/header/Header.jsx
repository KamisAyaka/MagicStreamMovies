import Button from "react-bootstrap/Button";
import Container from "react-bootstrap/Container";
import Nav from "react-bootstrap/Nav";
import Navbar from "react-bootstrap/Navbar";
import { useNavigate, NavLink, Link } from "react-router-dom";
import useAuth from "../../hooks/useAuth";

const Header = ({ handleLogout }) => {
  const navigate = useNavigate();
  const { auth } = useAuth();
  return (
    <Navbar
      bg="dark"
      variant="dark"
      expand="lg"
      sticky="top"
      className="shadow-sm"
    >
      <Container>
        <Navbar.Brand>MagicStreamMovies</Navbar.Brand>

        <Navbar.Toggle aria-controls="main-navbar-nav" />
        <Navbar.Collapse>
          <Nav className="ms-auto">
            <Nav.Link as={NavLink} to="/">
              Home
            </Nav.Link>
            <Nav.Link as={NavLink} to="/recommended">
              Recommended
            </Nav.Link>
          </Nav>

          <Nav className="ms-auto align-items-center">
            {auth ? (
              <>
                <span className="me-3 text-light">
                  Hello,<strong>{auth.first_name}</strong>
                </span>
                <Button
                  variant="outline-light"
                  size="sm"
                  onClick={handleLogout}
                >
                  Logout
                </Button>
              </>
            ) : (
              <>
                <Button
                  variant="outline-info"
                  size="sm"
                  className="me-2"
                  onClick={() => navigate("/login")}
                >
                  Login
                </Button>
                <Button
                  variant="info"
                  size="sm"
                  onClick={() => navigate("/register")}
                >
                  Register
                </Button>
              </>
            )}
          </Nav>
        </Navbar.Collapse>
      </Container>
    </Navbar>
  );
};
export default Header;
