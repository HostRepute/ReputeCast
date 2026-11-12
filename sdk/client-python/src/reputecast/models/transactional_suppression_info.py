from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_suppression_info_scope import TransactionalSuppressionInfoScope
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalSuppressionInfo")



@_attrs_define
class TransactionalSuppressionInfo:
    """ 
        Attributes:
            reason (str):  Example: hard_bounce.
            since (datetime.datetime):
            source (str):  Example: Postmark.
            code (str | Unset):  Example: 550 5.1.1.
            detail (str | Unset):
            id (str | Unset): The suppression as it is now; empty once it was removed
            scope (TransactionalSuppressionInfoScope | Unset):
     """

    reason: str
    since: datetime.datetime
    source: str
    code: str | Unset = UNSET
    detail: str | Unset = UNSET
    id: str | Unset = UNSET
    scope: TransactionalSuppressionInfoScope | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        reason = self.reason

        since = self.since.isoformat()

        source = self.source

        code = self.code

        detail = self.detail

        id = self.id

        scope: str | Unset = UNSET
        if not isinstance(self.scope, Unset):
            scope = self.scope.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "reason": reason,
            "since": since,
            "source": source,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if detail is not UNSET:
            field_dict["detail"] = detail
        if id is not UNSET:
            field_dict["id"] = id
        if scope is not UNSET:
            field_dict["scope"] = scope

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reason = d.pop("reason")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        source = d.pop("source")

        code = d.pop("code", UNSET)

        detail = d.pop("detail", UNSET)

        id = d.pop("id", UNSET)

        _scope = d.pop("scope", UNSET)
        scope: TransactionalSuppressionInfoScope | Unset
        if isinstance(_scope,  Unset):
            scope = UNSET
        else:
            scope = TransactionalSuppressionInfoScope(_scope)




        transactional_suppression_info = cls(
            reason=reason,
            since=since,
            source=source,
            code=code,
            detail=detail,
            id=id,
            scope=scope,
        )

        return transactional_suppression_info

