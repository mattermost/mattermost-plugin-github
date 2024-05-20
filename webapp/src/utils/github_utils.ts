export function isUrlCanPreview(url: string) {
    if (url.includes('github.com/') && url.split('github.com/')[1]) {
        const [owner, repo, type, number] = url.split('github.com/')[1].split('/');
        return Boolean(owner && repo && type && number);
    }
    return false;
}
