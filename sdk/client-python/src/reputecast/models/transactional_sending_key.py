from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_sending_key_mode import TransactionalSendingKeyMode
from ..models.transactional_sending_key_preset import TransactionalSendingKeyPreset
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalSendingKey")



@_attrs_define
class TransactionalSendingKey:
    """ 
        Attributes:
            categories (list[str]): Category names the key is limited to
            created_at (datetime.datetime):
            created_by (str):
            expired (bool):
            id (str):
            masked (str):
            mode (TransactionalSendingKeyMode):
            name (str):
            preset (TransactionalSendingKeyPreset):
            rate_per_second (int):
            templates (list[str]):
            test_addresses (list[str]):
            expires_at (datetime.datetime | Unset):
            last_used_at (datetime.datetime | Unset):
            last_used_ip (str | Unset):
     """

    categories: list[str]
    created_at: datetime.datetime
    created_by: str
    expired: bool
    id: str
    masked: str
    mode: TransactionalSendingKeyMode
    name: str
    preset: TransactionalSendingKeyPreset
    rate_per_second: int
    templates: list[str]
    test_addresses: list[str]
    expires_at: datetime.datetime | Unset = UNSET
    last_used_at: datetime.datetime | Unset = UNSET
    last_used_ip: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        categories = self.categories



        created_at = self.created_at.isoformat()

        created_by = self.created_by

        expired = self.expired

        id = self.id

        masked = self.masked

        mode = self.mode.value

        name = self.name

        preset = self.preset.value

        rate_per_second = self.rate_per_second

        templates = self.templates



        test_addresses = self.test_addresses



        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        last_used_at: str | Unset = UNSET
        if not isinstance(self.last_used_at, Unset):
            last_used_at = self.last_used_at.isoformat()

        last_used_ip = self.last_used_ip


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "categories": categories,
            "createdAt": created_at,
            "createdBy": created_by,
            "expired": expired,
            "id": id,
            "masked": masked,
            "mode": mode,
            "name": name,
            "preset": preset,
            "ratePerSecond": rate_per_second,
            "templates": templates,
            "testAddresses": test_addresses,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if last_used_at is not UNSET:
            field_dict["lastUsedAt"] = last_used_at
        if last_used_ip is not UNSET:
            field_dict["lastUsedIp"] = last_used_ip

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        categories = cast(list[str], d.pop("categories"))


        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        expired = d.pop("expired")

        id = d.pop("id")

        masked = d.pop("masked")

        mode = TransactionalSendingKeyMode(d.pop("mode"))




        name = d.pop("name")

        preset = TransactionalSendingKeyPreset(d.pop("preset"))




        rate_per_second = d.pop("ratePerSecond")

        templates = cast(list[str], d.pop("templates"))


        test_addresses = cast(list[str], d.pop("testAddresses"))


        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _last_used_at = d.pop("lastUsedAt", UNSET)
        last_used_at: datetime.datetime | Unset
        if isinstance(_last_used_at,  Unset):
            last_used_at = UNSET
        else:
            last_used_at = datetime.datetime.fromisoformat(_last_used_at)




        last_used_ip = d.pop("lastUsedIp", UNSET)

        transactional_sending_key = cls(
            categories=categories,
            created_at=created_at,
            created_by=created_by,
            expired=expired,
            id=id,
            masked=masked,
            mode=mode,
            name=name,
            preset=preset,
            rate_per_second=rate_per_second,
            templates=templates,
            test_addresses=test_addresses,
            expires_at=expires_at,
            last_used_at=last_used_at,
            last_used_ip=last_used_ip,
        )

        return transactional_sending_key

