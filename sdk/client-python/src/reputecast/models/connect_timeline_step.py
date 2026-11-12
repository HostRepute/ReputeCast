from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_timeline_step_key import ConnectTimelineStepKey
from ..models.connect_timeline_step_status import ConnectTimelineStepStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectTimelineStep")



@_attrs_define
class ConnectTimelineStep:
    """ 
        Attributes:
            detail (str):
            key (ConnectTimelineStepKey):
            status (ConnectTimelineStepStatus):
            title (str):
            at (datetime.datetime | Unset):
     """

    detail: str
    key: ConnectTimelineStepKey
    status: ConnectTimelineStepStatus
    title: str
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        detail = self.detail

        key = self.key.value

        status = self.status.value

        title = self.title

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "key": key,
            "status": status,
            "title": title,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detail = d.pop("detail")

        key = ConnectTimelineStepKey(d.pop("key"))




        status = ConnectTimelineStepStatus(d.pop("status"))




        title = d.pop("title")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        connect_timeline_step = cls(
            detail=detail,
            key=key,
            status=status,
            title=title,
            at=at,
        )

        return connect_timeline_step

