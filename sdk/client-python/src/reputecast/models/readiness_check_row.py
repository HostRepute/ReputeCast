from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ReadinessCheckRow")



@_attrs_define
class ReadinessCheckRow:
    """ 
        Attributes:
            count (int):
            detail (str):
            label (str):
            gets (str | Unset):
            gets_name (str | Unset):
            language (str | Unset):
     """

    count: int
    detail: str
    label: str
    gets: str | Unset = UNSET
    gets_name: str | Unset = UNSET
    language: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        detail = self.detail

        label = self.label

        gets = self.gets

        gets_name = self.gets_name

        language = self.language


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "detail": detail,
            "label": label,
        })
        if gets is not UNSET:
            field_dict["gets"] = gets
        if gets_name is not UNSET:
            field_dict["getsName"] = gets_name
        if language is not UNSET:
            field_dict["language"] = language

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        detail = d.pop("detail")

        label = d.pop("label")

        gets = d.pop("gets", UNSET)

        gets_name = d.pop("getsName", UNSET)

        language = d.pop("language", UNSET)

        readiness_check_row = cls(
            count=count,
            detail=detail,
            label=label,
            gets=gets,
            gets_name=gets_name,
            language=language,
        )

        return readiness_check_row

