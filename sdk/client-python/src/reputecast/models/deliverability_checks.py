from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_checks_interval import DeliverabilityChecksInterval
from ..models.deliverability_checks_via import DeliverabilityChecksVia
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.deliverability_check_run import DeliverabilityCheckRun





T = TypeVar("T", bound="DeliverabilityChecks")



@_attrs_define
class DeliverabilityChecks:
    """ 
        Attributes:
            builtin_sources (int):
            interval (DeliverabilityChecksInterval):
            via (DeliverabilityChecksVia):
            last_at (datetime.datetime | Unset):
            next_at (datetime.datetime | Unset):
            running (DeliverabilityCheckRun | Unset):
     """

    builtin_sources: int
    interval: DeliverabilityChecksInterval
    via: DeliverabilityChecksVia
    last_at: datetime.datetime | Unset = UNSET
    next_at: datetime.datetime | Unset = UNSET
    running: DeliverabilityCheckRun | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_check_run import DeliverabilityCheckRun # noqa: PLC0415
        builtin_sources = self.builtin_sources

        interval = self.interval.value

        via = self.via.value

        last_at: str | Unset = UNSET
        if not isinstance(self.last_at, Unset):
            last_at = self.last_at.isoformat()

        next_at: str | Unset = UNSET
        if not isinstance(self.next_at, Unset):
            next_at = self.next_at.isoformat()

        running: dict[str, Any] | Unset = UNSET
        if not isinstance(self.running, Unset):
            running = self.running.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "builtinSources": builtin_sources,
            "interval": interval,
            "via": via,
        })
        if last_at is not UNSET:
            field_dict["lastAt"] = last_at
        if next_at is not UNSET:
            field_dict["nextAt"] = next_at
        if running is not UNSET:
            field_dict["running"] = running

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_check_run import DeliverabilityCheckRun # noqa: PLC0415
        d = dict(src_dict)
        builtin_sources = d.pop("builtinSources")

        interval = DeliverabilityChecksInterval(d.pop("interval"))




        via = DeliverabilityChecksVia(d.pop("via"))




        _last_at = d.pop("lastAt", UNSET)
        last_at: datetime.datetime | Unset
        if isinstance(_last_at,  Unset):
            last_at = UNSET
        else:
            last_at = datetime.datetime.fromisoformat(_last_at)




        _next_at = d.pop("nextAt", UNSET)
        next_at: datetime.datetime | Unset
        if isinstance(_next_at,  Unset):
            next_at = UNSET
        else:
            next_at = datetime.datetime.fromisoformat(_next_at)




        _running = d.pop("running", UNSET)
        running: DeliverabilityCheckRun | Unset
        if isinstance(_running,  Unset):
            running = UNSET
        else:
            running = DeliverabilityCheckRun.from_dict(_running)




        deliverability_checks = cls(
            builtin_sources=builtin_sources,
            interval=interval,
            via=via,
            last_at=last_at,
            next_at=next_at,
            running=running,
        )

        return deliverability_checks

