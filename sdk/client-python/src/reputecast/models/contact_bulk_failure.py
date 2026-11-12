from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_bulk_failure_code import ContactBulkFailureCode






T = TypeVar("T", bound="ContactBulkFailure")



@_attrs_define
class ContactBulkFailure:
    """ 
        Attributes:
            code (ContactBulkFailureCode):
            email (str):
            reason (str):
     """

    code: ContactBulkFailureCode
    email: str
    reason: str





    def to_dict(self) -> dict[str, Any]:
        code = self.code.value

        email = self.email

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "email": email,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = ContactBulkFailureCode(d.pop("code"))




        email = d.pop("email")

        reason = d.pop("reason")

        contact_bulk_failure = cls(
            code=code,
            email=email,
            reason=reason,
        )

        return contact_bulk_failure

