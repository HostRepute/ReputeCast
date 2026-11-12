from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_detail_status import ImportDetailStatus
from ..models.import_detail_step import ImportDetailStep
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.import_connection import ImportConnection
  from ..models.import_error import ImportError_
  from ..models.import_log_entry import ImportLogEntry
  from ..models.import_options import ImportOptions
  from ..models.import_progress import ImportProgress
  from ..models.import_provider_pulled import ImportProviderPulled
  from ..models.import_stats import ImportStats
  from ..models.import_summary import ImportSummary





T = TypeVar("T", bound="ImportDetail")



@_attrs_define
class ImportDetail:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by (str):
            field_count (int):
            file_name (str):
            has_credentials (bool): An API key is stored (it is never returned)
            id (str):
            log (list[ImportLogEntry]):
            options (ImportOptions):
            progress (ImportProgress):
            provider_sync (list[ImportProviderPulled]):
            reports_cleared (bool):
            source (str):
            source_name (str):
            staged_rows (int):
            status (ImportDetailStatus):
            step (ImportDetailStep):
            connection (ImportConnection | Unset):
            error (ImportError_ | Unset):
            finished_at (datetime.datetime | Unset):
            previous (ImportSummary | Unset):
            reports_until (datetime.datetime | Unset):
            staged_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            stats (ImportStats | Unset):
     """

    created_at: datetime.datetime
    created_by: str
    field_count: int
    file_name: str
    has_credentials: bool
    id: str
    log: list[ImportLogEntry]
    options: ImportOptions
    progress: ImportProgress
    provider_sync: list[ImportProviderPulled]
    reports_cleared: bool
    source: str
    source_name: str
    staged_rows: int
    status: ImportDetailStatus
    step: ImportDetailStep
    connection: ImportConnection | Unset = UNSET
    error: ImportError_ | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    previous: ImportSummary | Unset = UNSET
    reports_until: datetime.datetime | Unset = UNSET
    staged_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    stats: ImportStats | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_connection import ImportConnection # noqa: PLC0415
        from ..models.import_error import ImportError_ # noqa: PLC0415
        from ..models.import_log_entry import ImportLogEntry # noqa: PLC0415
        from ..models.import_options import ImportOptions # noqa: PLC0415
        from ..models.import_progress import ImportProgress # noqa: PLC0415
        from ..models.import_provider_pulled import ImportProviderPulled # noqa: PLC0415
        from ..models.import_stats import ImportStats # noqa: PLC0415
        from ..models.import_summary import ImportSummary # noqa: PLC0415
        created_at = self.created_at.isoformat()

        created_by = self.created_by

        field_count = self.field_count

        file_name = self.file_name

        has_credentials = self.has_credentials

        id = self.id

        log = []
        for log_item_data in self.log:
            log_item = log_item_data.to_dict()
            log.append(log_item)



        options = self.options.to_dict()

        progress = self.progress.to_dict()

        provider_sync = []
        for provider_sync_item_data in self.provider_sync:
            provider_sync_item = provider_sync_item_data.to_dict()
            provider_sync.append(provider_sync_item)



        reports_cleared = self.reports_cleared

        source = self.source

        source_name = self.source_name

        staged_rows = self.staged_rows

        status = self.status.value

        step = self.step.value

        connection: dict[str, Any] | Unset = UNSET
        if not isinstance(self.connection, Unset):
            connection = self.connection.to_dict()

        error: dict[str, Any] | Unset = UNSET
        if not isinstance(self.error, Unset):
            error = self.error.to_dict()

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        previous: dict[str, Any] | Unset = UNSET
        if not isinstance(self.previous, Unset):
            previous = self.previous.to_dict()

        reports_until: str | Unset = UNSET
        if not isinstance(self.reports_until, Unset):
            reports_until = self.reports_until.isoformat()

        staged_at: str | Unset = UNSET
        if not isinstance(self.staged_at, Unset):
            staged_at = self.staged_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        stats: dict[str, Any] | Unset = UNSET
        if not isinstance(self.stats, Unset):
            stats = self.stats.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdBy": created_by,
            "fieldCount": field_count,
            "fileName": file_name,
            "hasCredentials": has_credentials,
            "id": id,
            "log": log,
            "options": options,
            "progress": progress,
            "providerSync": provider_sync,
            "reportsCleared": reports_cleared,
            "source": source,
            "sourceName": source_name,
            "stagedRows": staged_rows,
            "status": status,
            "step": step,
        })
        if connection is not UNSET:
            field_dict["connection"] = connection
        if error is not UNSET:
            field_dict["error"] = error
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if previous is not UNSET:
            field_dict["previous"] = previous
        if reports_until is not UNSET:
            field_dict["reportsUntil"] = reports_until
        if staged_at is not UNSET:
            field_dict["stagedAt"] = staged_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if stats is not UNSET:
            field_dict["stats"] = stats

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_connection import ImportConnection # noqa: PLC0415
        from ..models.import_error import ImportError_ # noqa: PLC0415
        from ..models.import_log_entry import ImportLogEntry # noqa: PLC0415
        from ..models.import_options import ImportOptions # noqa: PLC0415
        from ..models.import_progress import ImportProgress # noqa: PLC0415
        from ..models.import_provider_pulled import ImportProviderPulled # noqa: PLC0415
        from ..models.import_stats import ImportStats # noqa: PLC0415
        from ..models.import_summary import ImportSummary # noqa: PLC0415
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        field_count = d.pop("fieldCount")

        file_name = d.pop("fileName")

        has_credentials = d.pop("hasCredentials")

        id = d.pop("id")

        log = []
        _log = d.pop("log")
        for log_item_data in (_log):
            log_item = ImportLogEntry.from_dict(log_item_data)



            log.append(log_item)


        options = ImportOptions.from_dict(d.pop("options"))




        progress = ImportProgress.from_dict(d.pop("progress"))




        provider_sync = []
        _provider_sync = d.pop("providerSync")
        for provider_sync_item_data in (_provider_sync):
            provider_sync_item = ImportProviderPulled.from_dict(provider_sync_item_data)



            provider_sync.append(provider_sync_item)


        reports_cleared = d.pop("reportsCleared")

        source = d.pop("source")

        source_name = d.pop("sourceName")

        staged_rows = d.pop("stagedRows")

        status = ImportDetailStatus(d.pop("status"))




        step = ImportDetailStep(d.pop("step"))




        _connection = d.pop("connection", UNSET)
        connection: ImportConnection | Unset
        if isinstance(_connection,  Unset):
            connection = UNSET
        else:
            connection = ImportConnection.from_dict(_connection)




        _error = d.pop("error", UNSET)
        error: ImportError_ | Unset
        if isinstance(_error,  Unset):
            error = UNSET
        else:
            error = ImportError_.from_dict(_error)




        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        _previous = d.pop("previous", UNSET)
        previous: ImportSummary | Unset
        if isinstance(_previous,  Unset):
            previous = UNSET
        else:
            previous = ImportSummary.from_dict(_previous)




        _reports_until = d.pop("reportsUntil", UNSET)
        reports_until: datetime.datetime | Unset
        if isinstance(_reports_until,  Unset):
            reports_until = UNSET
        else:
            reports_until = datetime.datetime.fromisoformat(_reports_until)




        _staged_at = d.pop("stagedAt", UNSET)
        staged_at: datetime.datetime | Unset
        if isinstance(_staged_at,  Unset):
            staged_at = UNSET
        else:
            staged_at = datetime.datetime.fromisoformat(_staged_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        _stats = d.pop("stats", UNSET)
        stats: ImportStats | Unset
        if isinstance(_stats,  Unset):
            stats = UNSET
        else:
            stats = ImportStats.from_dict(_stats)




        import_detail = cls(
            created_at=created_at,
            created_by=created_by,
            field_count=field_count,
            file_name=file_name,
            has_credentials=has_credentials,
            id=id,
            log=log,
            options=options,
            progress=progress,
            provider_sync=provider_sync,
            reports_cleared=reports_cleared,
            source=source,
            source_name=source_name,
            staged_rows=staged_rows,
            status=status,
            step=step,
            connection=connection,
            error=error,
            finished_at=finished_at,
            previous=previous,
            reports_until=reports_until,
            staged_at=staged_at,
            started_at=started_at,
            stats=stats,
        )

        return import_detail

