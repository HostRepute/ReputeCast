from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_ip_source_from import ProviderIPSourceFrom
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderIPSource")



@_attrs_define
class ProviderIPSource:
    """ 
        Attributes:
            editable (bool):
            from_ (ProviderIPSourceFrom):
            checked_at (datetime.datetime | Unset):
            error (str | Unset):
            error_at (datetime.datetime | Unset):
            set_at (datetime.datetime | Unset):
            set_by (str | Unset):
     """

    editable: bool
    from_: ProviderIPSourceFrom
    checked_at: datetime.datetime | Unset = UNSET
    error: str | Unset = UNSET
    error_at: datetime.datetime | Unset = UNSET
    set_at: datetime.datetime | Unset = UNSET
    set_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        editable = self.editable

        from_ = self.from_.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        error = self.error

        error_at: str | Unset = UNSET
        if not isinstance(self.error_at, Unset):
            error_at = self.error_at.isoformat()

        set_at: str | Unset = UNSET
        if not isinstance(self.set_at, Unset):
            set_at = self.set_at.isoformat()

        set_by = self.set_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "editable": editable,
            "from": from_,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if error is not UNSET:
            field_dict["error"] = error
        if error_at is not UNSET:
            field_dict["errorAt"] = error_at
        if set_at is not UNSET:
            field_dict["setAt"] = set_at
        if set_by is not UNSET:
            field_dict["setBy"] = set_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        editable = d.pop("editable")

        from_ = ProviderIPSourceFrom(d.pop("from"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        error = d.pop("error", UNSET)

        _error_at = d.pop("errorAt", UNSET)
        error_at: datetime.datetime | Unset
        if isinstance(_error_at,  Unset):
            error_at = UNSET
        else:
            error_at = datetime.datetime.fromisoformat(_error_at)




        _set_at = d.pop("setAt", UNSET)
        set_at: datetime.datetime | Unset
        if isinstance(_set_at,  Unset):
            set_at = UNSET
        else:
            set_at = datetime.datetime.fromisoformat(_set_at)




        set_by = d.pop("setBy", UNSET)

        provider_ip_source = cls(
            editable=editable,
            from_=from_,
            checked_at=checked_at,
            error=error,
            error_at=error_at,
            set_at=set_at,
            set_by=set_by,
        )

        return provider_ip_source

