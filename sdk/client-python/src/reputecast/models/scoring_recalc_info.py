from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.scoring_recalc_info_reason import ScoringRecalcInfoReason
from ..models.scoring_recalc_info_status import ScoringRecalcInfoStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.scoring_changes import ScoringChanges
  from ..models.scoring_recalc_summary import ScoringRecalcSummary





T = TypeVar("T", bound="ScoringRecalcInfo")



@_attrs_define
class ScoringRecalcInfo:
    """ 
        Attributes:
            changes (ScoringChanges):
            done (int):
            id (str):
            label (str):
            reason (ScoringRecalcInfoReason):
            start_automations (bool):
            started_at (datetime.datetime):
            started_by (str):
            status (ScoringRecalcInfoStatus):
            total (int):
            failed_at (datetime.datetime | Unset):
            finished_at (datetime.datetime | Unset):
            remaining_seconds (int | Unset):
            summary (ScoringRecalcSummary | Unset):
     """

    changes: ScoringChanges
    done: int
    id: str
    label: str
    reason: ScoringRecalcInfoReason
    start_automations: bool
    started_at: datetime.datetime
    started_by: str
    status: ScoringRecalcInfoStatus
    total: int
    failed_at: datetime.datetime | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    remaining_seconds: int | Unset = UNSET
    summary: ScoringRecalcSummary | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_changes import ScoringChanges # noqa: PLC0415
        from ..models.scoring_recalc_summary import ScoringRecalcSummary # noqa: PLC0415
        changes = self.changes.to_dict()

        done = self.done

        id = self.id

        label = self.label

        reason = self.reason.value

        start_automations = self.start_automations

        started_at = self.started_at.isoformat()

        started_by = self.started_by

        status = self.status.value

        total = self.total

        failed_at: str | Unset = UNSET
        if not isinstance(self.failed_at, Unset):
            failed_at = self.failed_at.isoformat()

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        remaining_seconds = self.remaining_seconds

        summary: dict[str, Any] | Unset = UNSET
        if not isinstance(self.summary, Unset):
            summary = self.summary.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "done": done,
            "id": id,
            "label": label,
            "reason": reason,
            "startAutomations": start_automations,
            "startedAt": started_at,
            "startedBy": started_by,
            "status": status,
            "total": total,
        })
        if failed_at is not UNSET:
            field_dict["failedAt"] = failed_at
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if remaining_seconds is not UNSET:
            field_dict["remainingSeconds"] = remaining_seconds
        if summary is not UNSET:
            field_dict["summary"] = summary

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_changes import ScoringChanges # noqa: PLC0415
        from ..models.scoring_recalc_summary import ScoringRecalcSummary # noqa: PLC0415
        d = dict(src_dict)
        changes = ScoringChanges.from_dict(d.pop("changes"))




        done = d.pop("done")

        id = d.pop("id")

        label = d.pop("label")

        reason = ScoringRecalcInfoReason(d.pop("reason"))




        start_automations = d.pop("startAutomations")

        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        started_by = d.pop("startedBy")

        status = ScoringRecalcInfoStatus(d.pop("status"))




        total = d.pop("total")

        _failed_at = d.pop("failedAt", UNSET)
        failed_at: datetime.datetime | Unset
        if isinstance(_failed_at,  Unset):
            failed_at = UNSET
        else:
            failed_at = datetime.datetime.fromisoformat(_failed_at)




        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        remaining_seconds = d.pop("remainingSeconds", UNSET)

        _summary = d.pop("summary", UNSET)
        summary: ScoringRecalcSummary | Unset
        if isinstance(_summary,  Unset):
            summary = UNSET
        else:
            summary = ScoringRecalcSummary.from_dict(_summary)




        scoring_recalc_info = cls(
            changes=changes,
            done=done,
            id=id,
            label=label,
            reason=reason,
            start_automations=start_automations,
            started_at=started_at,
            started_by=started_by,
            status=status,
            total=total,
            failed_at=failed_at,
            finished_at=finished_at,
            remaining_seconds=remaining_seconds,
            summary=summary,
        )

        return scoring_recalc_info

