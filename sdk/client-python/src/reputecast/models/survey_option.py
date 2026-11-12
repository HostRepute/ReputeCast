from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SurveyOption")



@_attrs_define
class SurveyOption:
    """ 
        Attributes:
            key (int):
            label (str):
            tag (str | Unset):
     """

    key: int
    label: str
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
        })
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        tag = d.pop("tag", UNSET)

        survey_option = cls(
            key=key,
            label=label,
            tag=tag,
        )

        return survey_option

