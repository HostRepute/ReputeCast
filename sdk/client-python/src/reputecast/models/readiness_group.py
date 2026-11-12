from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_group_key import ReadinessGroupKey
from ..models.readiness_group_status import ReadinessGroupStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReadinessGroup")



@_attrs_define
class ReadinessGroup:
    """ 
        Attributes:
            caption (str):
            key (ReadinessGroupKey):
            passed (int):
            running (str):
            status (ReadinessGroupStatus):
            summary (str):
            title (str):
            total (int):
            note (str | Unset):
     """

    caption: str
    key: ReadinessGroupKey
    passed: int
    running: str
    status: ReadinessGroupStatus
    summary: str
    title: str
    total: int
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        caption = self.caption

        key = self.key.value

        passed = self.passed

        running = self.running

        status = self.status.value

        summary = self.summary

        title = self.title

        total = self.total

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "caption": caption,
            "key": key,
            "passed": passed,
            "running": running,
            "status": status,
            "summary": summary,
            "title": title,
            "total": total,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        caption = d.pop("caption")

        key = ReadinessGroupKey(d.pop("key"))




        passed = d.pop("passed")

        running = d.pop("running")

        status = ReadinessGroupStatus(d.pop("status"))




        summary = d.pop("summary")

        title = d.pop("title")

        total = d.pop("total")

        note = d.pop("note", UNSET)

        readiness_group = cls(
            caption=caption,
            key=key,
            passed=passed,
            running=running,
            status=status,
            summary=summary,
            title=title,
            total=total,
            note=note,
        )

        return readiness_group

