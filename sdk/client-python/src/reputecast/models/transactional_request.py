from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalRequest")



@_attrs_define
class TransactionalRequest:
    """ 
        Attributes:
            body (Any):
            method (str):
            path (str):
            smtp (bool):
            idempotency_key (str | Unset):
     """

    body: Any
    method: str
    path: str
    smtp: bool
    idempotency_key: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        body = self.body

        method = self.method

        path = self.path

        smtp = self.smtp

        idempotency_key = self.idempotency_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "method": method,
            "path": path,
            "smtp": smtp,
        })
        if idempotency_key is not UNSET:
            field_dict["idempotencyKey"] = idempotency_key

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        body = d.pop("body")

        method = d.pop("method")

        path = d.pop("path")

        smtp = d.pop("smtp")

        idempotency_key = d.pop("idempotencyKey", UNSET)

        transactional_request = cls(
            body=body,
            method=method,
            path=path,
            smtp=smtp,
            idempotency_key=idempotency_key,
        )

        return transactional_request

