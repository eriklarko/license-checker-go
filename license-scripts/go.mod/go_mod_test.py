import unittest
import os
import sys
import tempfile
from unittest.mock import call, patch, MagicMock

# Add the parent directory to the Python path to import go_mod
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..")))
import go_mod


class TestGoMod(unittest.TestCase):

    def setUp(self):
        self.temp_dir = tempfile.mkdtemp()
        self.go_mod_content = "module github.com/example/project\n\ngo 1.16\n"
        self.go_mod_path = os.path.join(self.temp_dir, "go.mod")
        with open(self.go_mod_path, "w") as f:
            f.write(self.go_mod_content)

    def tearDown(self):
        os.remove(self.go_mod_path)
        os.rmdir(self.temp_dir)

    def test_parse_arguments(self):
        with patch(
            "sys.argv",
            [
                "go_mod.py",
                "--go-licenses-path",
                "/custom/path",
                "--go-mod-file",
                "/custom/go.mod",
            ],
        ):
            args = go_mod.parse_arguments()
            self.assertEqual(args.go_licenses_path, "/custom/path")
            self.assertEqual(args.go_mod_file, "/custom/go.mod")

    def test_assert_go_mod_file_exists__existing_file(self):
        go_mod.assert_go_mod_file_exists(self.go_mod_path)

    def test_assert_go_mod_file_exists__nonexistent_file(self):
        with self.assertRaises(SystemExit):
            go_mod.assert_go_mod_file_exists("/nonexistent/path")

    def test_get_package_name(self):
        package_name = go_mod.get_package_name(self.go_mod_path)
        self.assertEqual(package_name, "github.com/example/project")

    @patch("subprocess.run")
    def test_run_go_licenses(self, mock_run):
        mock_run.return_value = MagicMock(
            stdout="dep1,url1,MIT\ndep2,url2,Apache-2.0\n"
        )

        output = go_mod.run_go_licenses(
            "/path/to/go-licenses", "github.com/example/project"
        )

        self.assertEqual(output, "dep1,url1,MIT\ndep2,url2,Apache-2.0\n")

    @patch("go_mod.assert_go_mod_file_exists")
    @patch("shutil.which")
    @patch("go_mod.get_package_name")
    @patch("go_mod.run_go_licenses")
    @patch("go_mod.install_go_licenses")
    def test_main__happy_path(
        self,
        mock_run_go_licenses,
        mock_get_package_name,
        mock_which,
        mock_assert_go_mod_file_exists,
        mock_install_go_licenses,
    ):
        # Simulate the presence of go-licenses in the system path
        mock_which.return_value = "/path/to/go-licenses"

        # Simulate a package name that would be extracted from go.mod
        mock_get_package_name.return_value = "github.com/example/project"

        # Simulate the output of running go-licenses
        mock_run_go_licenses.return_value = "dep1,url1,MIT\ndep2,url2,Apache-2.0\n"


        # Run the main function and capture its print output
        # This allows us to verify what would be printed to the console
        with patch("builtins.print") as mock_print:
            go_mod.main()

            # Verify that the correct output was printed
            mock_print.assert_has_calls([
                call("dep1: MIT"),
                call("dep2: Apache-2.0")
            ], any_order=True)

            # Check that nothing unexpected was printed
            self.assertEqual(mock_print.call_count, 2, "Unexpected additional output was printed")

        # Assert it checked if the go.mod file exists
        mock_assert_go_mod_file_exists.assert_called_once_with("go.mod")

        # Assert it didn't try to install go-licenses
        mock_install_go_licenses.assert_not_called()

        # Assert it called the correct go-licenses command
        mock_run_go_licenses.assert_called_once_with("go-licenses", "github.com/example/project")


    @patch("go_mod.parse_arguments")
    def test_main__no_go_mod_file(
        self,
        mock_parse_arguments,
    ):
        mock_parse_arguments.return_value = MagicMock(
            go_mod_file="/nonexistent/go.mod"
        )

        with self.assertRaises(SystemExit):
            go_mod.main()

    @patch("go_mod.run_go_licenses")
    @patch("go_mod.install_go_licenses")
    @patch("go_mod.assert_go_mod_file_exists")
    @patch("go_mod.get_package_name")
    @patch("shutil.which")
    def test_main__go_licenses_not_installed(
        self,
        mock_run_go_licenses,
        mock_install_go_licenses,
        mock_get_package_name,
        mock_which,
        _,
    ):
        # Simulate the absence of go-licenses in the system path
        mock_which.return_value = None

        # Simulate the output of running go-licenses
        mock_run_go_licenses.return_value = "dep1,url1,MIT\ndep2,url2,Apache-2.0\n"

        mock_get_package_name.return_value = "github.com/example/project"

        with patch("builtins.print") as mock_print:
            go_mod.main()

            mock_print.assert_has_calls([
                call("dep1: MIT"),
                call("dep2: Apache-2.0")
            ], any_order=True)

        # check that it tried to install go-licenses
        mock_install_go_licenses.assert_called_once()

    @patch("go_mod.run_go_licenses")
    @patch("go_mod.install_go_licenses")
    @patch("shutil.which")
    @patch("go_mod.parse_arguments")
    def test_main__not_installed__no_install_flag(
        self,
        mock_parse_arguments,
        mock_which,
        mock_install_go_licenses,
        mock_run_go_licenses,
    ):
        # Specify the path to go-licenses and set no-install flag
        mock_parse_arguments.return_value = MagicMock(
            go_licenses_path="/path/to/go-licenses",
            no_install=True
        )

        # Simulate the absence of go-licenses in the system path
        mock_which.return_value = None

        # Expect SystemExit when go-licenses is not found and --no-install is set
        with self.assertRaises(SystemExit) as cm:
            go_mod.main()

        self.assertEqual(cm.exception.code, 1)

        # Check that it didn't try to install go-licenses
        mock_install_go_licenses.assert_not_called()

        # Check that it didn't try to run go-licenses
        mock_run_go_licenses.assert_not_called()

    @patch("go_mod.run_go_licenses")
    @patch("go_mod.install_go_licenses")
    @patch("shutil.which")
    @patch("go_mod.parse_arguments")
    @patch.dict(os.environ, {"CI": "true"})
    def test_main__not_installed__ci_environment(
        self,
        mock_parse_arguments,
        mock_which,
        mock_install_go_licenses,
        mock_run_go_licenses,
    ):
        # In CI environment, no_install should be True by default
        os.environ["CI"] = "1"
        mock_parse_arguments.return_value = MagicMock(
            go_licenses_path="/path/to/go-licenses",
            go_mod_file="./go.mod",
        )

        # Simulate the absence of go-licenses in the system path
        mock_which.return_value = None

        # Expect SystemExit when go-licenses is not found in CI environment
        with self.assertRaises(SystemExit) as cm:
            go_mod.main()

        self.assertEqual(cm.exception.code, 1)

        # Check that it didn't try to install go-licenses
        mock_install_go_licenses.assert_not_called()

        # Check that it didn't try to run go-licenses
        mock_run_go_licenses.assert_not_called()


if __name__ == "__main__":
    unittest.main()
