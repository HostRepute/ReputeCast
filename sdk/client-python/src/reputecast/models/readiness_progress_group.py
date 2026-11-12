from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_progress_group_key import ReadinessProgressGroupKey
from ..models.readiness_progress_group_status import ReadinessProgressGroupStatus
from ..models.readiness_progress_group_unit import ReadinessProgressGroupUnit
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReadinessProgressGroup")



@_attrs_define
class ReadinessProgressGroup:
    """ 
        Attributes:
            checked (int):
            count (int):
            done (int):
            key (ReadinessProgressGroupKey):
            of (int):
            status (ReadinessProgressGroupStatus):
            text (str):
            title (str):
            total (int):
            unit (ReadinessProgressGroupUnit | Unset):
     """

    checked: int
    count: int
    done: int
    key: ReadinessProgressGroupKey
    of: int
    status: ReadinessProgressGroupStatus
    text: str
    title: str
    total: int
    unit: ReadinessProgressGroupUnit | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checked = self.checked

        count = self.count

        done = self.done

        key = self.key.value

        of = self.of

        status = self.status.value

        text = self.text

        title = self.title

        total = self.total

        unit: str | Unset = UNSET
        if not isinstance(self.unit, Unset):
            unit = self.unit.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checked": checked,
            "count": count,
            "done": done,
            "key": key,
            "of": of,
            "status": status,
            "text": text,
            "title": title,
            "total": total,
        })
        if unit is not UNSET:
            field_dict["unit"] = unit

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checked = d.pop("checked")

        count = d.pop("count")

        done = d.pop("done")

        key = ReadinessProgressGroupKey(d.pop("key"))




        of = d.pop("of")

        status = ReadinessProgressGroupStatus(d.pop("status"))




        text = d.pop("text")

        title = d.pop("title")

        total = d.pop("total")

        _unit = d.pop("unit", UNSET)
        unit: ReadinessProgressGroupUnit | Unset
        if isinstance(_unit,  Unset):
            unit = UNSET
        else:
            unit = ReadinessProgressGroupUnit(_unit)




        readiness_progress_group = cls(
            checked=checked,
            count=count,
            done=done,
            key=key,
            of=of,
            status=status,
            text=text,
            title=title,
            total=total,
            unit=unit,
        )

        return readiness_progress_group

