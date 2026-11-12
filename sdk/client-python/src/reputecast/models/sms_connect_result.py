from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.account import Account
  from ..models.sms_found_sender import SmsFoundSender
  from ..models.sms_webhook_setup import SmsWebhookSetup





T = TypeVar("T", bound="SmsConnectResult")



@_attrs_define
class SmsConnectResult:
    """ 
        Attributes:
            account (Account):
            provider_id (str):
            senders (list[SmsFoundSender]):
            setup_id (str):
            webhooks (SmsWebhookSetup):
     """

    account: Account
    provider_id: str
    senders: list[SmsFoundSender]
    setup_id: str
    webhooks: SmsWebhookSetup





    def to_dict(self) -> dict[str, Any]:
        from ..models.account import Account # noqa: PLC0415
        from ..models.sms_found_sender import SmsFoundSender # noqa: PLC0415
        from ..models.sms_webhook_setup import SmsWebhookSetup # noqa: PLC0415
        account = self.account.to_dict()

        provider_id = self.provider_id

        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        setup_id = self.setup_id

        webhooks = self.webhooks.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "providerId": provider_id,
            "senders": senders,
            "setupId": setup_id,
            "webhooks": webhooks,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.account import Account # noqa: PLC0415
        from ..models.sms_found_sender import SmsFoundSender # noqa: PLC0415
        from ..models.sms_webhook_setup import SmsWebhookSetup # noqa: PLC0415
        d = dict(src_dict)
        account = Account.from_dict(d.pop("account"))




        provider_id = d.pop("providerId")

        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = SmsFoundSender.from_dict(senders_item_data)



            senders.append(senders_item)


        setup_id = d.pop("setupId")

        webhooks = SmsWebhookSetup.from_dict(d.pop("webhooks"))




        sms_connect_result = cls(
            account=account,
            provider_id=provider_id,
            senders=senders,
            setup_id=setup_id,
            webhooks=webhooks,
        )

        return sms_connect_result

