from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VisibilityCondition")



@_attrs_define
class VisibilityCondition:
    """ 
        Attributes:
            kind (str):
            op (str):
            field (str | Unset):
            label (str | Unset):
            to (str | Unset):
            value (str | Unset):
     """

    kind: str
    op: str
    field: str | Unset = UNSET
    label: str | Unset = UNSET
    to: str | Unset = UNSET
    value: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        op = self.op

        field = self.field

        label = self.label

        to = self.to

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "op": op,
        })
        if field is not UNSET:
            field_dict["field"] = field
        if label is not UNSET:
            field_dict["label"] = label
        if to is not UNSET:
            field_dict["to"] = to
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        op = d.pop("op")

        field = d.pop("field", UNSET)

        label = d.pop("label", UNSET)

        to = d.pop("to", UNSET)

        value = d.pop("value", UNSET)

        visibility_condition = cls(
            kind=kind,
            op=op,
            field=field,
            label=label,
            to=to,
            value=value,
        )

        return visibility_condition

