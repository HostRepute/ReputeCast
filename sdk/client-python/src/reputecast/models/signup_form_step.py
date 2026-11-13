from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SignupFormStep")



@_attrs_define
class SignupFormStep:
    """ 
        Attributes:
            fields (list[str]):
            id (str): s1 to s4, kept when steps move so their translations stay
            button (str | Unset):
            description (str | Unset):
            heading (str | Unset):
     """

    fields: list[str]
    id: str
    button: str | Unset = UNSET
    description: str | Unset = UNSET
    heading: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        fields = self.fields



        id = self.id

        button = self.button

        description = self.description

        heading = self.heading


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fields": fields,
            "id": id,
        })
        if button is not UNSET:
            field_dict["button"] = button
        if description is not UNSET:
            field_dict["description"] = description
        if heading is not UNSET:
            field_dict["heading"] = heading

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fields = cast(list[str], d.pop("fields"))


        id = d.pop("id")

        button = d.pop("button", UNSET)

        description = d.pop("description", UNSET)

        heading = d.pop("heading", UNSET)

        signup_form_step = cls(
            fields=fields,
            id=id,
            button=button,
            description=description,
            heading=heading,
        )

        return signup_form_step

