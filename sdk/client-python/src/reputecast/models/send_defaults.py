from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SendDefaults")



@_attrs_define
class SendDefaults:
    """ 
        Attributes:
            exclude_unengaged (bool):
            typed_multiple (float):
     """

    exclude_unengaged: bool
    typed_multiple: float





    def to_dict(self) -> dict[str, Any]:
        exclude_unengaged = self.exclude_unengaged

        typed_multiple = self.typed_multiple


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "excludeUnengaged": exclude_unengaged,
            "typedMultiple": typed_multiple,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        exclude_unengaged = d.pop("excludeUnengaged")

        typed_multiple = d.pop("typedMultiple")

        send_defaults = cls(
            exclude_unengaged=exclude_unengaged,
            typed_multiple=typed_multiple,
        )

        return send_defaults

