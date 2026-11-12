from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_filter import ContactFilter





T = TypeVar("T", bound="ContactExportInput")



@_attrs_define
class ContactExportInput:
    """ 
        Attributes:
            all_matching (bool | Unset):
            filter_ (ContactFilter | Unset):
            ids (list[str] | Unset):
     """

    all_matching: bool | Unset = UNSET
    filter_: ContactFilter | Unset = UNSET
    ids: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        all_matching = self.all_matching

        filter_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.filter_, Unset):
            filter_ = self.filter_.to_dict()

        ids: list[str] | Unset = UNSET
        if not isinstance(self.ids, Unset):
            ids = self.ids




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if all_matching is not UNSET:
            field_dict["allMatching"] = all_matching
        if filter_ is not UNSET:
            field_dict["filter"] = filter_
        if ids is not UNSET:
            field_dict["ids"] = ids

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        d = dict(src_dict)
        all_matching = d.pop("allMatching", UNSET)

        _filter_ = d.pop("filter", UNSET)
        filter_: ContactFilter | Unset
        if isinstance(_filter_,  Unset):
            filter_ = UNSET
        else:
            filter_ = ContactFilter.from_dict(_filter_)




        ids = cast(list[str], d.pop("ids", UNSET))


        contact_export_input = cls(
            all_matching=all_matching,
            filter_=filter_,
            ids=ids,
        )

        return contact_export_input

