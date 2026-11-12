from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SuppressionSource")



@_attrs_define
class SuppressionSource:
    """ 
        Attributes:
            kind (str):
            label (str): Amazon SES, Preference center, Linh Nguyen, Brevo import…
            adapter (str | Unset):
     """

    kind: str
    label: str
    adapter: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        label = self.label

        adapter = self.adapter


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if adapter is not UNSET:
            field_dict["adapter"] = adapter

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        label = d.pop("label")

        adapter = d.pop("adapter", UNSET)

        suppression_source = cls(
            kind=kind,
            label=label,
            adapter=adapter,
        )

        return suppression_source

