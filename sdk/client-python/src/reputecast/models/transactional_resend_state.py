from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_resend_state_reason import TransactionalResendStateReason
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="TransactionalResendState")



@_attrs_define
class TransactionalResendState:
    """ 
        Attributes:
            available (bool):
            reason (TransactionalResendStateReason | Unset):
            sensitive (list[str] | Unset):
     """

    available: bool
    reason: TransactionalResendStateReason | Unset = UNSET
    sensitive: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        sensitive: list[str] | Unset = UNSET
        if not isinstance(self.sensitive, Unset):
            sensitive = self.sensitive




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason
        if sensitive is not UNSET:
            field_dict["sensitive"] = sensitive

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        _reason = d.pop("reason", UNSET)
        reason: TransactionalResendStateReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = TransactionalResendStateReason(_reason)




        sensitive = cast(list[str], d.pop("sensitive", UNSET))


        transactional_resend_state = cls(
            available=available,
            reason=reason,
            sensitive=sensitive,
        )

        return transactional_resend_state

