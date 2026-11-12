from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.apply_input_kind import ApplyInputKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="ApplyInput")



@_attrs_define
class ApplyInput:
    """ 
        Attributes:
            from_ (str):
            kind (ApplyInputKind):
            to (str):
            block (str | Unset):
     """

    from_: str
    kind: ApplyInputKind
    to: str
    block: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from_ = self.from_

        kind = self.kind.value

        to = self.to

        block = self.block


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "kind": kind,
            "to": to,
        })
        if block is not UNSET:
            field_dict["block"] = block

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_ = d.pop("from")

        kind = ApplyInputKind(d.pop("kind"))




        to = d.pop("to")

        block = d.pop("block", UNSET)

        apply_input = cls(
            from_=from_,
            kind=kind,
            to=to,
            block=block,
        )

        return apply_input

