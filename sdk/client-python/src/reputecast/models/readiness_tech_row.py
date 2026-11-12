from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ReadinessTechRow")



@_attrs_define
class ReadinessTechRow:
    """ 
        Attributes:
            label (str):
            value (str):
            before (str | Unset):
     """

    label: str
    value: str
    before: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        value = self.value

        before = self.before


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "value": value,
        })
        if before is not UNSET:
            field_dict["before"] = before

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        value = d.pop("value")

        before = d.pop("before", UNSET)

        readiness_tech_row = cls(
            label=label,
            value=value,
            before=before,
        )

        return readiness_tech_row

