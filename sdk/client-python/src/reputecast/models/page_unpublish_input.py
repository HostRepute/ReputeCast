from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PageUnpublishInput")



@_attrs_define
class PageUnpublishInput:
    """ 
        Attributes:
            reason (str | Unset):
            redirect_to (str | Unset):
     """

    reason: str | Unset = UNSET
    redirect_to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        reason = self.reason

        redirect_to = self.redirect_to


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if reason is not UNSET:
            field_dict["reason"] = reason
        if redirect_to is not UNSET:
            field_dict["redirectTo"] = redirect_to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reason = d.pop("reason", UNSET)

        redirect_to = d.pop("redirectTo", UNSET)

        page_unpublish_input = cls(
            reason=reason,
            redirect_to=redirect_to,
        )

        return page_unpublish_input

