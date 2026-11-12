from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_webhooks_silent_broadcast_policy import ProviderWebhooksSilentBroadcastPolicy
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.provider_awaiting_message import ProviderAwaitingMessage
  from ..models.provider_named_count import ProviderNamedCount
  from ..models.provider_webhook_check import ProviderWebhookCheck





T = TypeVar("T", bound="ProviderWebhooksSilent")



@_attrs_define
class ProviderWebhooksSilent:
    """ 
        Attributes:
            accepted (int):
            awaiting (list[ProviderAwaitingMessage]):
            broadcast_policy (ProviderWebhooksSilentBroadcastPolicy):
            broadcast_share_held (bool):
            campaigns (list[ProviderNamedCount]):
            can_backfill (bool):
            can_register (bool):
            can_send_test (bool):
            causes (list[str]):
            checks (list[ProviderWebhookCheck]):
            hourly (list[int]):
            hourly_from (datetime.datetime):
            in_grace (int):
            overdue (int):
            provider (str):
            provider_id (str):
            short (str):
            since (datetime.datetime):
            webhook_url (str):
     """

    accepted: int
    awaiting: list[ProviderAwaitingMessage]
    broadcast_policy: ProviderWebhooksSilentBroadcastPolicy
    broadcast_share_held: bool
    campaigns: list[ProviderNamedCount]
    can_backfill: bool
    can_register: bool
    can_send_test: bool
    causes: list[str]
    checks: list[ProviderWebhookCheck]
    hourly: list[int]
    hourly_from: datetime.datetime
    in_grace: int
    overdue: int
    provider: str
    provider_id: str
    short: str
    since: datetime.datetime
    webhook_url: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_awaiting_message import ProviderAwaitingMessage # noqa: PLC0415
        from ..models.provider_named_count import ProviderNamedCount # noqa: PLC0415
        from ..models.provider_webhook_check import ProviderWebhookCheck # noqa: PLC0415
        accepted = self.accepted

        awaiting = []
        for awaiting_item_data in self.awaiting:
            awaiting_item = awaiting_item_data.to_dict()
            awaiting.append(awaiting_item)



        broadcast_policy = self.broadcast_policy.value

        broadcast_share_held = self.broadcast_share_held

        campaigns = []
        for campaigns_item_data in self.campaigns:
            campaigns_item = campaigns_item_data.to_dict()
            campaigns.append(campaigns_item)



        can_backfill = self.can_backfill

        can_register = self.can_register

        can_send_test = self.can_send_test

        causes = self.causes



        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        hourly = self.hourly



        hourly_from = self.hourly_from.isoformat()

        in_grace = self.in_grace

        overdue = self.overdue

        provider = self.provider

        provider_id = self.provider_id

        short = self.short

        since = self.since.isoformat()

        webhook_url = self.webhook_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "awaiting": awaiting,
            "broadcastPolicy": broadcast_policy,
            "broadcastShareHeld": broadcast_share_held,
            "campaigns": campaigns,
            "canBackfill": can_backfill,
            "canRegister": can_register,
            "canSendTest": can_send_test,
            "causes": causes,
            "checks": checks,
            "hourly": hourly,
            "hourlyFrom": hourly_from,
            "inGrace": in_grace,
            "overdue": overdue,
            "provider": provider,
            "providerId": provider_id,
            "short": short,
            "since": since,
            "webhookUrl": webhook_url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_awaiting_message import ProviderAwaitingMessage # noqa: PLC0415
        from ..models.provider_named_count import ProviderNamedCount # noqa: PLC0415
        from ..models.provider_webhook_check import ProviderWebhookCheck # noqa: PLC0415
        d = dict(src_dict)
        accepted = d.pop("accepted")

        awaiting = []
        _awaiting = d.pop("awaiting")
        for awaiting_item_data in (_awaiting):
            awaiting_item = ProviderAwaitingMessage.from_dict(awaiting_item_data)



            awaiting.append(awaiting_item)


        broadcast_policy = ProviderWebhooksSilentBroadcastPolicy(d.pop("broadcastPolicy"))




        broadcast_share_held = d.pop("broadcastShareHeld")

        campaigns = []
        _campaigns = d.pop("campaigns")
        for campaigns_item_data in (_campaigns):
            campaigns_item = ProviderNamedCount.from_dict(campaigns_item_data)



            campaigns.append(campaigns_item)


        can_backfill = d.pop("canBackfill")

        can_register = d.pop("canRegister")

        can_send_test = d.pop("canSendTest")

        causes = cast(list[str], d.pop("causes"))


        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = ProviderWebhookCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        hourly = cast(list[int], d.pop("hourly"))


        hourly_from = datetime.datetime.fromisoformat(d.pop("hourlyFrom"))




        in_grace = d.pop("inGrace")

        overdue = d.pop("overdue")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        webhook_url = d.pop("webhookUrl")

        provider_webhooks_silent = cls(
            accepted=accepted,
            awaiting=awaiting,
            broadcast_policy=broadcast_policy,
            broadcast_share_held=broadcast_share_held,
            campaigns=campaigns,
            can_backfill=can_backfill,
            can_register=can_register,
            can_send_test=can_send_test,
            causes=causes,
            checks=checks,
            hourly=hourly,
            hourly_from=hourly_from,
            in_grace=in_grace,
            overdue=overdue,
            provider=provider,
            provider_id=provider_id,
            short=short,
            since=since,
            webhook_url=webhook_url,
        )

        return provider_webhooks_silent

