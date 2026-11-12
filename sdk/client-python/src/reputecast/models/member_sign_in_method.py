from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.member_sign_in_method_kind import MemberSignInMethodKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="MemberSignInMethod")



@_attrs_define
class MemberSignInMethod:
    """ 
        Attributes:
            kind (MemberSignInMethodKind):
            label (str):
            preset (str | Unset):
     """

    kind: MemberSignInMethodKind
    label: str
    preset: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        preset = self.preset


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if preset is not UNSET:
            field_dict["preset"] = preset

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = MemberSignInMethodKind(d.pop("kind"))




        label = d.pop("label")

        preset = d.pop("preset", UNSET)

        member_sign_in_method = cls(
            kind=kind,
            label=label,
            preset=preset,
        )

        return member_sign_in_method

