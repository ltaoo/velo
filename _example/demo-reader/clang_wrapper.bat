@echo off
setlocal enabledelayedexpansion
set args=
for %%a in (%*) do (
    if not "%%a"=="-mthreads" (
        set "args=!args! %%a"
    )
)
clang++ !args!
