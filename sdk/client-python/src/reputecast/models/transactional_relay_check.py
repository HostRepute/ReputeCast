from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_relay_check_key import TransactionalRelayCheckKey






T = TypeVar("T", bound="TransactionalRelayCheck")



@_attrs_define
class TransactionalRelayCheck:
    """ 
        Attributes:
            detail (str):
            key (TransactionalRelayCheckKey):
            ok (bool):
     """

    detail: str
    key: TransactionalRelayCheckKey
    ok: bool





    def to_dict(self) -> dict[str, Any]:
        detail = self.detail

        key = self.key.value

        ok = self.ok


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "key": key,
            "ok": ok,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detail = d.pop("detail")

        key = TransactionalRelayCheckKey(d.pop("key"))




        ok = d.pop("ok")

        transactional_relay_check = cls(
            detail=detail,
            key=key,
            ok=ok,
        )

        return transactional_relay_check

