from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="IncidentLogLine")



@_attrs_define
class IncidentLogLine:
    """ 
        Attributes:
            at (datetime.datetime):
            bad (bool):
            text (str):
     """

    at: datetime.datetime
    bad: bool
    text: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        bad = self.bad

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "bad": bad,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        bad = d.pop("bad")

        text = d.pop("text")

        incident_log_line = cls(
            at=at,
            bad=bad,
            text=text,
        )

        return incident_log_line

