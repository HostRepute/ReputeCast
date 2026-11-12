from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.push_step_stats_skipped import PushStepStatsSkipped





T = TypeVar("T", bound="PushStepStats")



@_attrs_define
class PushStepStats:
    """ 
        Attributes:
            clicks (int):
            node_id (str):
            sent (int):
            shown (int):
            skipped (PushStepStatsSkipped):
            waiting (int):
            first_at (datetime.datetime | Unset):
            last_at (datetime.datetime | Unset):
     """

    clicks: int
    node_id: str
    sent: int
    shown: int
    skipped: PushStepStatsSkipped
    waiting: int
    first_at: datetime.datetime | Unset = UNSET
    last_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_step_stats_skipped import PushStepStatsSkipped # noqa: PLC0415
        clicks = self.clicks

        node_id = self.node_id

        sent = self.sent

        shown = self.shown

        skipped = self.skipped.to_dict()

        waiting = self.waiting

        first_at: str | Unset = UNSET
        if not isinstance(self.first_at, Unset):
            first_at = self.first_at.isoformat()

        last_at: str | Unset = UNSET
        if not isinstance(self.last_at, Unset):
            last_at = self.last_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "nodeId": node_id,
            "sent": sent,
            "shown": shown,
            "skipped": skipped,
            "waiting": waiting,
        })
        if first_at is not UNSET:
            field_dict["firstAt"] = first_at
        if last_at is not UNSET:
            field_dict["lastAt"] = last_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_step_stats_skipped import PushStepStatsSkipped # noqa: PLC0415
        d = dict(src_dict)
        clicks = d.pop("clicks")

        node_id = d.pop("nodeId")

        sent = d.pop("sent")

        shown = d.pop("shown")

        skipped = PushStepStatsSkipped.from_dict(d.pop("skipped"))




        waiting = d.pop("waiting")

        _first_at = d.pop("firstAt", UNSET)
        first_at: datetime.datetime | Unset
        if isinstance(_first_at,  Unset):
            first_at = UNSET
        else:
            first_at = datetime.datetime.fromisoformat(_first_at)




        _last_at = d.pop("lastAt", UNSET)
        last_at: datetime.datetime | Unset
        if isinstance(_last_at,  Unset):
            last_at = UNSET
        else:
            last_at = datetime.datetime.fromisoformat(_last_at)




        push_step_stats = cls(
            clicks=clicks,
            node_id=node_id,
            sent=sent,
            shown=shown,
            skipped=skipped,
            waiting=waiting,
            first_at=first_at,
            last_at=last_at,
        )

        return push_step_stats

