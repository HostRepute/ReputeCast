from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_webhook_check_key import ProviderWebhookCheckKey
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderWebhookCheck")



@_attrs_define
class ProviderWebhookCheck:
    """ 
        Attributes:
            key (ProviderWebhookCheckKey):
            at (datetime.datetime | Unset):
            count (int | Unset):
            detail (str | Unset):
            id (str | Unset):
            ok (bool | Unset):
            queued (int | Unset):
            url (str | Unset):
     """

    key: ProviderWebhookCheckKey
    at: datetime.datetime | Unset = UNSET
    count: int | Unset = UNSET
    detail: str | Unset = UNSET
    id: str | Unset = UNSET
    ok: bool | Unset = UNSET
    queued: int | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        count = self.count

        detail = self.detail

        id = self.id

        ok = self.ok

        queued = self.queued

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if count is not UNSET:
            field_dict["count"] = count
        if detail is not UNSET:
            field_dict["detail"] = detail
        if id is not UNSET:
            field_dict["id"] = id
        if ok is not UNSET:
            field_dict["ok"] = ok
        if queued is not UNSET:
            field_dict["queued"] = queued
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = ProviderWebhookCheckKey(d.pop("key"))




        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        count = d.pop("count", UNSET)

        detail = d.pop("detail", UNSET)

        id = d.pop("id", UNSET)

        ok = d.pop("ok", UNSET)

        queued = d.pop("queued", UNSET)

        url = d.pop("url", UNSET)

        provider_webhook_check = cls(
            key=key,
            at=at,
            count=count,
            detail=detail,
            id=id,
            ok=ok,
            queued=queued,
            url=url,
        )

        return provider_webhook_check

