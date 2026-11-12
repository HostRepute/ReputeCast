from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_progress_started_by_type import ReadinessProgressStartedByType
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.readiness_progress_group import ReadinessProgressGroup





T = TypeVar("T", bound="ReadinessProgress")



@_attrs_define
class ReadinessProgress:
    """ 
        Attributes:
            done (int):
            groups (list[ReadinessProgressGroup]):
            queued (bool):
            run_id (str):
            started_at (datetime.datetime):
            total (int):
            trigger (str):
            started_by (str | Unset):
            started_by_type (ReadinessProgressStartedByType | Unset):
     """

    done: int
    groups: list[ReadinessProgressGroup]
    queued: bool
    run_id: str
    started_at: datetime.datetime
    total: int
    trigger: str
    started_by: str | Unset = UNSET
    started_by_type: ReadinessProgressStartedByType | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_progress_group import ReadinessProgressGroup # noqa: PLC0415
        done = self.done

        groups = []
        for groups_item_data in self.groups:
            groups_item = groups_item_data.to_dict()
            groups.append(groups_item)



        queued = self.queued

        run_id = self.run_id

        started_at = self.started_at.isoformat()

        total = self.total

        trigger = self.trigger

        started_by = self.started_by

        started_by_type: str | Unset = UNSET
        if not isinstance(self.started_by_type, Unset):
            started_by_type = self.started_by_type.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "done": done,
            "groups": groups,
            "queued": queued,
            "runId": run_id,
            "startedAt": started_at,
            "total": total,
            "trigger": trigger,
        })
        if started_by is not UNSET:
            field_dict["startedBy"] = started_by
        if started_by_type is not UNSET:
            field_dict["startedByType"] = started_by_type

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_progress_group import ReadinessProgressGroup # noqa: PLC0415
        d = dict(src_dict)
        done = d.pop("done")

        groups = []
        _groups = d.pop("groups")
        for groups_item_data in (_groups):
            groups_item = ReadinessProgressGroup.from_dict(groups_item_data)



            groups.append(groups_item)


        queued = d.pop("queued")

        run_id = d.pop("runId")

        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        total = d.pop("total")

        trigger = d.pop("trigger")

        started_by = d.pop("startedBy", UNSET)

        _started_by_type = d.pop("startedByType", UNSET)
        started_by_type: ReadinessProgressStartedByType | Unset
        if isinstance(_started_by_type,  Unset):
            started_by_type = UNSET
        else:
            started_by_type = ReadinessProgressStartedByType(_started_by_type)




        readiness_progress = cls(
            done=done,
            groups=groups,
            queued=queued,
            run_id=run_id,
            started_at=started_at,
            total=total,
            trigger=trigger,
            started_by=started_by,
            started_by_type=started_by_type,
        )

        return readiness_progress

