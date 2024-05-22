export function isUrlCanPreview(url: string) {
    const {hostname, pathname} = new URL(url);
    if (hostname.includes('github.com') && pathname.split('/')[1]) {
        const [_, owner, repo, type, number] = pathname.split('/');
        return Boolean(owner && repo && type && number);
    }
    return false;
}
