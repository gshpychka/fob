package main

import "fmt"

// shellFunction wraps the program in bash and zsh, and makes the profile picked in it the shell's AWS profile.
var shellFunction = fmt.Sprintf(`%[1]s() {
  if [ "$#" -gt 0 ]; then
    command %[1]s "$@"
    return
  fi
  local profile
  profile="$(command %[1]s)" && export AWS_PROFILE="$profile"
}
`, program)

// shellSetup is the line of a shell's startup file that defines the function.
var shellSetup = fmt.Sprintf(`eval "$(%s init)"`, program)
