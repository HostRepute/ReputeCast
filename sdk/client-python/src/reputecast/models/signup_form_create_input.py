from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_create_input_kind import SignupFormCreateInputKind






T = TypeVar("T", bound="SignupFormCreateInput")



@_attrs_define
class SignupFormCreateInput:
    """ 
        Attributes:
            kind (SignupFormCreateInputKind):
            name (str):
     """

    kind: SignupFormCreateInputKind
    name: str





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = SignupFormCreateInputKind(d.pop("kind"))




        name = d.pop("name")

        signup_form_create_input = cls(
            kind=kind,
            name=name,
        )

        return signup_form_create_input

