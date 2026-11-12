from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_error_code import TransactionalErrorCode
  from ..models.transactional_limits import TransactionalLimits
  from ..models.transactional_relay import TransactionalRelay
  from ..models.transactional_sender_option import TransactionalSenderOption
  from ..models.transactional_sending_key import TransactionalSendingKey





T = TypeVar("T", bound="TransactionalSetup")



@_attrs_define
class TransactionalSetup:
    """ 
        Attributes:
            base_url (str):
            can_manage_keys (bool):
            errors (list[TransactionalErrorCode]):
            headers (list[str]):
            key_helpers (list[str]):
            keys (list[TransactionalSendingKey]):
            limits (TransactionalLimits):
            openapi_url (str):
            relay (TransactionalRelay):
            senders (list[TransactionalSenderOption]):
     """

    base_url: str
    can_manage_keys: bool
    errors: list[TransactionalErrorCode]
    headers: list[str]
    key_helpers: list[str]
    keys: list[TransactionalSendingKey]
    limits: TransactionalLimits
    openapi_url: str
    relay: TransactionalRelay
    senders: list[TransactionalSenderOption]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_error_code import TransactionalErrorCode # noqa: PLC0415
        from ..models.transactional_limits import TransactionalLimits # noqa: PLC0415
        from ..models.transactional_relay import TransactionalRelay # noqa: PLC0415
        from ..models.transactional_sender_option import TransactionalSenderOption # noqa: PLC0415
        from ..models.transactional_sending_key import TransactionalSendingKey # noqa: PLC0415
        base_url = self.base_url

        can_manage_keys = self.can_manage_keys

        errors = []
        for errors_item_data in self.errors:
            errors_item = errors_item_data.to_dict()
            errors.append(errors_item)



        headers = self.headers



        key_helpers = self.key_helpers



        keys = []
        for keys_item_data in self.keys:
            keys_item = keys_item_data.to_dict()
            keys.append(keys_item)



        limits = self.limits.to_dict()

        openapi_url = self.openapi_url

        relay = self.relay.to_dict()

        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseUrl": base_url,
            "canManageKeys": can_manage_keys,
            "errors": errors,
            "headers": headers,
            "keyHelpers": key_helpers,
            "keys": keys,
            "limits": limits,
            "openapiUrl": openapi_url,
            "relay": relay,
            "senders": senders,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_error_code import TransactionalErrorCode # noqa: PLC0415
        from ..models.transactional_limits import TransactionalLimits # noqa: PLC0415
        from ..models.transactional_relay import TransactionalRelay # noqa: PLC0415
        from ..models.transactional_sender_option import TransactionalSenderOption # noqa: PLC0415
        from ..models.transactional_sending_key import TransactionalSendingKey # noqa: PLC0415
        d = dict(src_dict)
        base_url = d.pop("baseUrl")

        can_manage_keys = d.pop("canManageKeys")

        errors = []
        _errors = d.pop("errors")
        for errors_item_data in (_errors):
            errors_item = TransactionalErrorCode.from_dict(errors_item_data)



            errors.append(errors_item)


        headers = cast(list[str], d.pop("headers"))


        key_helpers = cast(list[str], d.pop("keyHelpers"))


        keys = []
        _keys = d.pop("keys")
        for keys_item_data in (_keys):
            keys_item = TransactionalSendingKey.from_dict(keys_item_data)



            keys.append(keys_item)


        limits = TransactionalLimits.from_dict(d.pop("limits"))




        openapi_url = d.pop("openapiUrl")

        relay = TransactionalRelay.from_dict(d.pop("relay"))




        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = TransactionalSenderOption.from_dict(senders_item_data)



            senders.append(senders_item)


        transactional_setup = cls(
            base_url=base_url,
            can_manage_keys=can_manage_keys,
            errors=errors,
            headers=headers,
            key_helpers=key_helpers,
            keys=keys,
            limits=limits,
            openapi_url=openapi_url,
            relay=relay,
            senders=senders,
        )

        return transactional_setup

