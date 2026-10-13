# fyne-cross's darwin image, with the frameworks of the macOS SDK where its
# linker looks for them.
#
# fyne-cross mounts the SDK at /sdk and links with "zig cc --sysroot /sdk
# -F/System/Library/Frameworks", but zig does not apply the sysroot to -F,
# so it looks in /System/Library/Frameworks of the container and finds no
# framework ("unable to find framework 'Security'"). The link points there.
FROM fyneio/fyne-cross-images:darwin

# The image has no /System, the rm is in case a later one does
RUN rm -rf /System/Library/Frameworks \
 && mkdir -p /System/Library \
 && ln -s /sdk/System/Library/Frameworks /System/Library/Frameworks
