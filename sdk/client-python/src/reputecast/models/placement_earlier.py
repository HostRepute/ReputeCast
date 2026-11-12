from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PlacementEarlier")



@_attrs_define
class PlacementEarlier:
    """ 
        Attributes:
            content_name (str):
            copies (int):
            inbox (int):
            started_at (datetime.datetime):
            test_id (str):
     """

    content_name: str
    copies: int
    inbox: int
    started_at: datetime.datetime
    test_id: str





    def to_dict(self) -> dict[str, Any]:
        content_name = self.content_name

        copies = self.copies

        inbox = self.inbox

        started_at = self.started_at.isoformat()

        test_id = self.test_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentName": content_name,
            "copies": copies,
            "inbox": inbox,
            "startedAt": started_at,
            "testId": test_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content_name = d.pop("contentName")

        copies = d.pop("copies")

        inbox = d.pop("inbox")

        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        test_id = d.pop("testId")

        placement_earlier = cls(
            content_name=content_name,
            copies=copies,
            inbox=inbox,
            started_at=started_at,
            test_id=test_id,
        )

        return placement_earlier

