from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_chip_kind import ConnectChipKind






T = TypeVar("T", bound="ConnectChip")



@_attrs_define
class ConnectChip:
    """ 
        Attributes:
            kind (ConnectChipKind):
            label (str):
     """

    kind: ConnectChipKind
    label: str





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = ConnectChipKind(d.pop("kind"))




        label = d.pop("label")

        connect_chip = cls(
            kind=kind,
            label=label,
        )

        return connect_chip

