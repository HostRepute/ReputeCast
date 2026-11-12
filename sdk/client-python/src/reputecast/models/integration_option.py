from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="IntegrationOption")



@_attrs_define
class IntegrationOption:
    """ 
        Attributes:
            id (str):
            name (str):
            detail (str | Unset):
     """

    id: str
    name: str
    detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        detail = self.detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        detail = d.pop("detail", UNSET)

        integration_option = cls(
            id=id,
            name=name,
            detail=detail,
        )

        return integration_option

