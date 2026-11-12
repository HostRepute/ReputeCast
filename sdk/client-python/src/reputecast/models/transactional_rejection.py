from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_rejection_reason import TransactionalRejectionReason
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_opt_out_info import TransactionalOptOutInfo
  from ..models.transactional_suppression_info import TransactionalSuppressionInfo





T = TypeVar("T", bound="TransactionalRejection")



@_attrs_define
class TransactionalRejection:
    """ 
        Attributes:
            reason (TransactionalRejectionReason):
            allowed (list[str] | Unset):
            error (str | Unset):
            opt_out (TransactionalOptOutInfo | Unset):
            suppression (TransactionalSuppressionInfo | Unset):
     """

    reason: TransactionalRejectionReason
    allowed: list[str] | Unset = UNSET
    error: str | Unset = UNSET
    opt_out: TransactionalOptOutInfo | Unset = UNSET
    suppression: TransactionalSuppressionInfo | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_opt_out_info import TransactionalOptOutInfo # noqa: PLC0415
        from ..models.transactional_suppression_info import TransactionalSuppressionInfo # noqa: PLC0415
        reason = self.reason.value

        allowed: list[str] | Unset = UNSET
        if not isinstance(self.allowed, Unset):
            allowed = self.allowed



        error = self.error

        opt_out: dict[str, Any] | Unset = UNSET
        if not isinstance(self.opt_out, Unset):
            opt_out = self.opt_out.to_dict()

        suppression: dict[str, Any] | Unset = UNSET
        if not isinstance(self.suppression, Unset):
            suppression = self.suppression.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "reason": reason,
        })
        if allowed is not UNSET:
            field_dict["allowed"] = allowed
        if error is not UNSET:
            field_dict["error"] = error
        if opt_out is not UNSET:
            field_dict["optOut"] = opt_out
        if suppression is not UNSET:
            field_dict["suppression"] = suppression

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_opt_out_info import TransactionalOptOutInfo # noqa: PLC0415
        from ..models.transactional_suppression_info import TransactionalSuppressionInfo # noqa: PLC0415
        d = dict(src_dict)
        reason = TransactionalRejectionReason(d.pop("reason"))




        allowed = cast(list[str], d.pop("allowed", UNSET))


        error = d.pop("error", UNSET)

        _opt_out = d.pop("optOut", UNSET)
        opt_out: TransactionalOptOutInfo | Unset
        if isinstance(_opt_out,  Unset):
            opt_out = UNSET
        else:
            opt_out = TransactionalOptOutInfo.from_dict(_opt_out)




        _suppression = d.pop("suppression", UNSET)
        suppression: TransactionalSuppressionInfo | Unset
        if isinstance(_suppression,  Unset):
            suppression = UNSET
        else:
            suppression = TransactionalSuppressionInfo.from_dict(_suppression)




        transactional_rejection = cls(
            reason=reason,
            allowed=allowed,
            error=error,
            opt_out=opt_out,
            suppression=suppression,
        )

        return transactional_rejection

