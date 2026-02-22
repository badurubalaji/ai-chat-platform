import { Pipe, PipeTransform } from '@angular/core';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';

@Pipe({
    name: 'markdownRender',
    standalone: true,
    pure: true,
})
export class MarkdownRenderPipe implements PipeTransform {
    constructor(private sanitizer: DomSanitizer) { }

    transform(value: string | null | undefined): SafeHtml {
        if (!value) {
            return '';
        }
        const html = this.parseMarkdown(value);
        return this.sanitizer.bypassSecurityTrustHtml(html);
    }

    private parseMarkdown(text: string): string {
        let html = this.escapeHtml(text);

        // Code blocks (```)
        html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_match, lang, code) => {
            return `<pre><code class="language-${lang}">${code.trim()}</code></pre>`;
        });

        // Inline code (`)
        html = html.replace(/`([^`]+)`/g, '<code>$1</code>');

        // Headers
        html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');
        html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>');
        html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>');

        // Bold (**text**)
        html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');

        // Italic (*text*)
        html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

        // Unordered lists
        html = html.replace(/^\s*[-*]\s+(.+)$/gm, '<li>$1</li>');
        html = html.replace(/(<li>.*<\/li>)/gs, '<ul>$1</ul>');
        // Clean up nested lists
        html = html.replace(/<\/ul>\s*<ul>/g, '');

        // Ordered lists
        html = html.replace(/^\s*\d+\.\s+(.+)$/gm, '<li>$1</li>');

        // Links
        html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');

        // Line breaks (double newline = paragraph)
        html = html.replace(/\n\n/g, '</p><p>');

        // Single newlines within paragraphs
        html = html.replace(/\n/g, '<br>');

        // Wrap in paragraph if not already wrapped
        if (!html.startsWith('<')) {
            html = `<p>${html}</p>`;
        }

        // Tables
        html = this.parseTables(html);

        return html;
    }

    private parseTables(html: string): string {
        // Simple table parsing for | col | col | format
        const tableRegex = /(<p>)?\|(.+)\|(<br>)\|[-| :]+\|((<br>)\|(.+)\|)+(<\/p>)?/g;
        return html.replace(tableRegex, (match) => {
            const rows = match.replace(/<\/?p>/g, '').split('<br>').filter(r => r.trim());
            if (rows.length < 2) return match;

            const headerCells = rows[0].split('|').filter(c => c.trim());
            let table = '<table><thead><tr>';
            headerCells.forEach(cell => {
                table += `<th>${cell.trim()}</th>`;
            });
            table += '</tr></thead><tbody>';

            // Skip header separator row (row[1])
            for (let i = 2; i < rows.length; i++) {
                const cells = rows[i].split('|').filter(c => c.trim());
                table += '<tr>';
                cells.forEach(cell => {
                    table += `<td>${cell.trim()}</td>`;
                });
                table += '</tr>';
            }
            table += '</tbody></table>';
            return table;
        });
        return html;
    }

    private escapeHtml(text: string): string {
        // Only escape angle brackets that aren't part of our markdown
        return text
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;');
    }
}
