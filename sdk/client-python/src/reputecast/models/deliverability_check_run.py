from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_check_run_status import DeliverabilityCheckRunStatus
from ..models.deliverability_check_run_trigger import DeliverabilityCheckRunTrigger
from ..models.deliverability_check_run_via import DeliverabilityCheckRunVia
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityCheckRun")



@_attrs_define
class DeliverabilityCheckRun:
    """ 
        Attributes:
            answered (int):
            assets (list[str]):
            id (str):
            listed (int):
            started_at (datetime.datetime):
            status (DeliverabilityCheckRunStatus):
            total (int):
            trigger (DeliverabilityCheckRunTrigger):
            via (DeliverabilityCheckRunVia):
            error (str | Unset):
            finished_at (datetime.datetime | Unset):
     """

    answered: int
    assets: list[str]
    id: str
    listed: int
    started_at: datetime.datetime
    status: DeliverabilityCheckRunStatus
    total: int
    trigger: DeliverabilityCheckRunTrigger
    via: DeliverabilityCheckRunVia
    error: str | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        answered = self.answered

        assets = self.assets



        id = self.id

        listed = self.listed

        started_at = self.started_at.isoformat()

        status = self.status.value

        total = self.total

        trigger = self.trigger.value

        via = self.via.value

        error = self.error

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answered": answered,
            "assets": assets,
            "id": id,
            "listed": listed,
            "startedAt": started_at,
            "status": status,
            "total": total,
            "trigger": trigger,
            "via": via,
        })
        if error is not UNSET:
            field_dict["error"] = error
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        answered = d.pop("answered")

        assets = cast(list[str], d.pop("assets"))


        id = d.pop("id")

        listed = d.pop("listed")

        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        status = DeliverabilityCheckRunStatus(d.pop("status"))




        total = d.pop("total")

        trigger = DeliverabilityCheckRunTrigger(d.pop("trigger"))




        via = DeliverabilityCheckRunVia(d.pop("via"))




        error = d.pop("error", UNSET)

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        deliverability_check_run = cls(
            answered=answered,
            assets=assets,
            id=id,
            listed=listed,
            started_at=started_at,
            status=status,
            total=total,
            trigger=trigger,
            via=via,
            error=error,
            finished_at=finished_at,
        )

        return deliverability_check_run

