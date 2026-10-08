# logger: unknown LOG_LEVEL silently becomes debug

`pkg/logger.New` maps any unrecognized value (`"warning"`, `"INFO"`, typos) to `slog.LevelDebug`, so a misconfigured production instance logs at the most verbose level without any warning. Consider `slog.Level.UnmarshalText` (case-insensitive, supports `warn`/`WARN+1`) and failing config load on invalid values.
