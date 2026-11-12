from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LeadAdsQuestion")



@_attrs_define
class LeadAdsQuestion:
    """ 
        Attributes:
            key (str):  Example: full_name.
            label (str):  Example: Full name.
            suggested (str):
            target (str):
            type_ (str): Meta's question type Example: FULL_NAME.
     """

    key: str
    label: str
    suggested: str
    target: str
    type_: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        suggested = self.suggested

        target = self.target

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "suggested": suggested,
            "target": target,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        suggested = d.pop("suggested")

        target = d.pop("target")

        type_ = d.pop("type")

        lead_ads_question = cls(
            key=key,
            label=label,
            suggested=suggested,
            target=target,
            type_=type_,
        )

        return lead_ads_question

