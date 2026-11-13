from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.integration_ai_facts import IntegrationAIFacts
  from ..models.integration_lead_facts import IntegrationLeadFacts
  from ..models.integration_payment_facts import IntegrationPaymentFacts
  from ..models.integration_store_facts import IntegrationStoreFacts





T = TypeVar("T", bound="IntegrationFacts")



@_attrs_define
class IntegrationFacts:
    """ 
        Attributes:
            actions (int):
            agents (int):
            endpoints (int):
            failing (int):
            mcp_enabled (bool):
            silent_sites (int):
            sites (int):
            triggers (int):
            updates (int):
            workflows (int):
            ai (IntegrationAIFacts | Unset):
            balance (int | Unset):
            key_created_at (datetime.datetime | Unset):
            last_call_at (datetime.datetime | Unset):
            last_sync_at (datetime.datetime | Unset):
            leads (IntegrationLeadFacts | Unset):
            payment (IntegrationPaymentFacts | Unset):
            price (float | Unset):
            store (IntegrationStoreFacts | Unset):
            team (str | Unset):
     """

    actions: int
    agents: int
    endpoints: int
    failing: int
    mcp_enabled: bool
    silent_sites: int
    sites: int
    triggers: int
    updates: int
    workflows: int
    ai: IntegrationAIFacts | Unset = UNSET
    balance: int | Unset = UNSET
    key_created_at: datetime.datetime | Unset = UNSET
    last_call_at: datetime.datetime | Unset = UNSET
    last_sync_at: datetime.datetime | Unset = UNSET
    leads: IntegrationLeadFacts | Unset = UNSET
    payment: IntegrationPaymentFacts | Unset = UNSET
    price: float | Unset = UNSET
    store: IntegrationStoreFacts | Unset = UNSET
    team: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_ai_facts import IntegrationAIFacts # noqa: PLC0415
        from ..models.integration_lead_facts import IntegrationLeadFacts # noqa: PLC0415
        from ..models.integration_payment_facts import IntegrationPaymentFacts # noqa: PLC0415
        from ..models.integration_store_facts import IntegrationStoreFacts # noqa: PLC0415
        actions = self.actions

        agents = self.agents

        endpoints = self.endpoints

        failing = self.failing

        mcp_enabled = self.mcp_enabled

        silent_sites = self.silent_sites

        sites = self.sites

        triggers = self.triggers

        updates = self.updates

        workflows = self.workflows

        ai: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ai, Unset):
            ai = self.ai.to_dict()

        balance = self.balance

        key_created_at: str | Unset = UNSET
        if not isinstance(self.key_created_at, Unset):
            key_created_at = self.key_created_at.isoformat()

        last_call_at: str | Unset = UNSET
        if not isinstance(self.last_call_at, Unset):
            last_call_at = self.last_call_at.isoformat()

        last_sync_at: str | Unset = UNSET
        if not isinstance(self.last_sync_at, Unset):
            last_sync_at = self.last_sync_at.isoformat()

        leads: dict[str, Any] | Unset = UNSET
        if not isinstance(self.leads, Unset):
            leads = self.leads.to_dict()

        payment: dict[str, Any] | Unset = UNSET
        if not isinstance(self.payment, Unset):
            payment = self.payment.to_dict()

        price = self.price

        store: dict[str, Any] | Unset = UNSET
        if not isinstance(self.store, Unset):
            store = self.store.to_dict()

        team = self.team


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "actions": actions,
            "agents": agents,
            "endpoints": endpoints,
            "failing": failing,
            "mcpEnabled": mcp_enabled,
            "silentSites": silent_sites,
            "sites": sites,
            "triggers": triggers,
            "updates": updates,
            "workflows": workflows,
        })
        if ai is not UNSET:
            field_dict["ai"] = ai
        if balance is not UNSET:
            field_dict["balance"] = balance
        if key_created_at is not UNSET:
            field_dict["keyCreatedAt"] = key_created_at
        if last_call_at is not UNSET:
            field_dict["lastCallAt"] = last_call_at
        if last_sync_at is not UNSET:
            field_dict["lastSyncAt"] = last_sync_at
        if leads is not UNSET:
            field_dict["leads"] = leads
        if payment is not UNSET:
            field_dict["payment"] = payment
        if price is not UNSET:
            field_dict["price"] = price
        if store is not UNSET:
            field_dict["store"] = store
        if team is not UNSET:
            field_dict["team"] = team

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_ai_facts import IntegrationAIFacts # noqa: PLC0415
        from ..models.integration_lead_facts import IntegrationLeadFacts # noqa: PLC0415
        from ..models.integration_payment_facts import IntegrationPaymentFacts # noqa: PLC0415
        from ..models.integration_store_facts import IntegrationStoreFacts # noqa: PLC0415
        d = dict(src_dict)
        actions = d.pop("actions")

        agents = d.pop("agents")

        endpoints = d.pop("endpoints")

        failing = d.pop("failing")

        mcp_enabled = d.pop("mcpEnabled")

        silent_sites = d.pop("silentSites")

        sites = d.pop("sites")

        triggers = d.pop("triggers")

        updates = d.pop("updates")

        workflows = d.pop("workflows")

        _ai = d.pop("ai", UNSET)
        ai: IntegrationAIFacts | Unset
        if isinstance(_ai,  Unset):
            ai = UNSET
        else:
            ai = IntegrationAIFacts.from_dict(_ai)




        balance = d.pop("balance", UNSET)

        _key_created_at = d.pop("keyCreatedAt", UNSET)
        key_created_at: datetime.datetime | Unset
        if isinstance(_key_created_at,  Unset):
            key_created_at = UNSET
        else:
            key_created_at = datetime.datetime.fromisoformat(_key_created_at)




        _last_call_at = d.pop("lastCallAt", UNSET)
        last_call_at: datetime.datetime | Unset
        if isinstance(_last_call_at,  Unset):
            last_call_at = UNSET
        else:
            last_call_at = datetime.datetime.fromisoformat(_last_call_at)




        _last_sync_at = d.pop("lastSyncAt", UNSET)
        last_sync_at: datetime.datetime | Unset
        if isinstance(_last_sync_at,  Unset):
            last_sync_at = UNSET
        else:
            last_sync_at = datetime.datetime.fromisoformat(_last_sync_at)




        _leads = d.pop("leads", UNSET)
        leads: IntegrationLeadFacts | Unset
        if isinstance(_leads,  Unset):
            leads = UNSET
        else:
            leads = IntegrationLeadFacts.from_dict(_leads)




        _payment = d.pop("payment", UNSET)
        payment: IntegrationPaymentFacts | Unset
        if isinstance(_payment,  Unset):
            payment = UNSET
        else:
            payment = IntegrationPaymentFacts.from_dict(_payment)




        price = d.pop("price", UNSET)

        _store = d.pop("store", UNSET)
        store: IntegrationStoreFacts | Unset
        if isinstance(_store,  Unset):
            store = UNSET
        else:
            store = IntegrationStoreFacts.from_dict(_store)




        team = d.pop("team", UNSET)

        integration_facts = cls(
            actions=actions,
            agents=agents,
            endpoints=endpoints,
            failing=failing,
            mcp_enabled=mcp_enabled,
            silent_sites=silent_sites,
            sites=sites,
            triggers=triggers,
            updates=updates,
            workflows=workflows,
            ai=ai,
            balance=balance,
            key_created_at=key_created_at,
            last_call_at=last_call_at,
            last_sync_at=last_sync_at,
            leads=leads,
            payment=payment,
            price=price,
            store=store,
            team=team,
        )

        return integration_facts

