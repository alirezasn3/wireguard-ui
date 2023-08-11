#!/bin/bash

wg syncconf $1 <(wg-quick strip $1)

