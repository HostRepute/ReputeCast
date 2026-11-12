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
  from ..models.readiness_ab_timeline import ReadinessABTimeline





T = TypeVar("T", bound="ReadinessTimeline")



@_attrs_define
class ReadinessTimeline:
    """ 
        Attributes:
            basis (list[str]):
            basis_label (str):
            end (datetime.datetime):
            start (datetime.datetime):
            ab (ReadinessABTimeline | Unset):
            best_time (bool | Unset):
            local_time (bool | Unset):
            open_ (bool | Unset):
     """

    basis: list[str]
    basis_label: str
    end: datetime.datetime
    start: datetime.datetime
    ab: ReadinessABTimeline | Unset = UNSET
    best_time: bool | Unset = UNSET
    local_time: bool | Unset = UNSET
    open_: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_ab_timeline import ReadinessABTimeline # noqa: PLC0415
        basis = self.basis



        basis_label = self.basis_label

        end = self.end.isoformat()

        start = self.start.isoformat()

        ab: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab, Unset):
            ab = self.ab.to_dict()

        best_time = self.best_time

        local_time = self.local_time

        open_ = self.open_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "basis": basis,
            "basisLabel": basis_label,
            "end": end,
            "start": start,
        })
        if ab is not UNSET:
            field_dict["ab"] = ab
        if best_time is not UNSET:
            field_dict["bestTime"] = best_time
        if local_time is not UNSET:
            field_dict["localTime"] = local_time
        if open_ is not UNSET:
            field_dict["open"] = open_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_ab_timeline import ReadinessABTimeline # noqa: PLC0415
        d = dict(src_dict)
        basis = cast(list[str], d.pop("basis"))


        basis_label = d.pop("basisLabel")

        end = datetime.datetime.fromisoformat(d.pop("end"))




        start = datetime.datetime.fromisoformat(d.pop("start"))




        _ab = d.pop("ab", UNSET)
        ab: ReadinessABTimeline | Unset
        if isinstance(_ab,  Unset):
            ab = UNSET
        else:
            ab = ReadinessABTimeline.from_dict(_ab)




        best_time = d.pop("bestTime", UNSET)

        local_time = d.pop("localTime", UNSET)

        open_ = d.pop("open", UNSET)

        readiness_timeline = cls(
            basis=basis,
            basis_label=basis_label,
            end=end,
            start=start,
            ab=ab,
            best_time=best_time,
            local_time=local_time,
            open_=open_,
        )

        return readiness_timeline

