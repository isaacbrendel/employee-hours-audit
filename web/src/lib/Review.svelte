<script lang="ts">
  import type { Suggestion } from "./api";
  import type { Column, ReviewRow } from "./types";
  import { columnLabel, columns } from "./types";

  let {
    rows,
    suggestEnabled,
    asking,
    suggestion,
    onSuggest,
    onUse,
    onDismiss,
  }: {
    rows: ReviewRow[];
    suggestEnabled: boolean;
    asking: string;
    suggestion: (Suggestion & { row: number }) | null;
    onSuggest: (row: ReviewRow, field: string, reason: string) => void;
    onUse: (row: number, field: Column, value: string) => void;
    onDismiss: () => void;
  } = $props();

  function key(row: number, field: string): string {
    return `${row}:${field}`;
  }
</script>

<div class="scroll">
  <table>
    <caption class="lede">
      Edit a row, then recheck it. It moves to clean data only when it passes.
      {#if suggestEnabled}A suggestion is not applied until you use the value and recheck.{/if}
    </caption>
    <thead>
      <tr>
        <th scope="col">Row</th>
        {#each columns as column (column)}
          <th scope="col">{columnLabel[column]}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each rows as item (item.row)}
        <tr>
          <th scope="row">{item.row}</th>
          {#each columns as column (column)}
            <td>
              <input
                type="text"
                aria-label={`${columnLabel[column]} for row ${item.row}`}
                bind:value={item.raw[column]}
              />
            </td>
          {/each}
        </tr>
        <tr>
          <td></td>
          <td colspan="6">
            <ul class="issues">
              {#each item.issues as issue (`${item.row}:${issue.field}:${issue.reason}`)}
                <li>
                  {issue.reason}
                  {#if suggestEnabled}
                    <div class="suggest">
                      <button
                        type="button"
                        disabled={asking === key(item.row, issue.field)}
                        onclick={() => onSuggest(item, issue.field, issue.reason)}
                      >
                        {asking === key(item.row, issue.field) ? "Asking…" : "Suggest a fix"}
                      </button>
                    </div>
                  {/if}
                  {#if suggestion && suggestion.row === item.row && suggestion.field === issue.field}
                    <div class="suggestion">
                      <p>{suggestion.explanation}</p>
                      <p>Suggested value: <strong>{suggestion.suggested}</strong></p>
                      <button
                        type="button"
                        onclick={() => onUse(item.row, issue.field as Column, suggestion.suggested)}
                      >Use this value</button>
                      <button type="button" class="secondary" onclick={onDismiss}>Leave it</button>
                    </div>
                  {/if}
                </li>
              {/each}
            </ul>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
