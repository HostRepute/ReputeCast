from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.approval_audience_rule import ApprovalAudienceRule
  from ..models.approval_client_signoff import ApprovalClientSignoff
  from ..models.approval_cost_rule import ApprovalCostRule





T = TypeVar("T", bound="ApprovalPolicy")



@_attrs_define
class ApprovalPolicy:
    """ 
        Attributes:
            approval_lists (bool):
            approvals_required (int):
            audience_min (ApprovalAudienceRule):
            author_lacks_send (bool):
            automation_publish (bool):
            emergency_bypass (bool):
            new_provider_days (int):
            post_publish (bool):
            ready_with_warnings (bool):
            transactional_publish (bool):
            warming_provider (bool):
            channel_cost (ApprovalCostRule | Unset):
            client_signoff (ApprovalClientSignoff | Unset):
     """

    approval_lists: bool
    approvals_required: int
    audience_min: ApprovalAudienceRule
    author_lacks_send: bool
    automation_publish: bool
    emergency_bypass: bool
    new_provider_days: int
    post_publish: bool
    ready_with_warnings: bool
    transactional_publish: bool
    warming_provider: bool
    channel_cost: ApprovalCostRule | Unset = UNSET
    client_signoff: ApprovalClientSignoff | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_audience_rule import ApprovalAudienceRule # noqa: PLC0415
        from ..models.approval_client_signoff import ApprovalClientSignoff # noqa: PLC0415
        from ..models.approval_cost_rule import ApprovalCostRule # noqa: PLC0415
        approval_lists = self.approval_lists

        approvals_required = self.approvals_required

        audience_min = self.audience_min.to_dict()

        author_lacks_send = self.author_lacks_send

        automation_publish = self.automation_publish

        emergency_bypass = self.emergency_bypass

        new_provider_days = self.new_provider_days

        post_publish = self.post_publish

        ready_with_warnings = self.ready_with_warnings

        transactional_publish = self.transactional_publish

        warming_provider = self.warming_provider

        channel_cost: dict[str, Any] | Unset = UNSET
        if not isinstance(self.channel_cost, Unset):
            channel_cost = self.channel_cost.to_dict()

        client_signoff: dict[str, Any] | Unset = UNSET
        if not isinstance(self.client_signoff, Unset):
            client_signoff = self.client_signoff.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvalLists": approval_lists,
            "approvalsRequired": approvals_required,
            "audienceMin": audience_min,
            "authorLacksSend": author_lacks_send,
            "automationPublish": automation_publish,
            "emergencyBypass": emergency_bypass,
            "newProviderDays": new_provider_days,
            "postPublish": post_publish,
            "readyWithWarnings": ready_with_warnings,
            "transactionalPublish": transactional_publish,
            "warmingProvider": warming_provider,
        })
        if channel_cost is not UNSET:
            field_dict["channelCost"] = channel_cost
        if client_signoff is not UNSET:
            field_dict["clientSignoff"] = client_signoff

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_audience_rule import ApprovalAudienceRule # noqa: PLC0415
        from ..models.approval_client_signoff import ApprovalClientSignoff # noqa: PLC0415
        from ..models.approval_cost_rule import ApprovalCostRule # noqa: PLC0415
        d = dict(src_dict)
        approval_lists = d.pop("approvalLists")

        approvals_required = d.pop("approvalsRequired")

        audience_min = ApprovalAudienceRule.from_dict(d.pop("audienceMin"))




        author_lacks_send = d.pop("authorLacksSend")

        automation_publish = d.pop("automationPublish")

        emergency_bypass = d.pop("emergencyBypass")

        new_provider_days = d.pop("newProviderDays")

        post_publish = d.pop("postPublish")

        ready_with_warnings = d.pop("readyWithWarnings")

        transactional_publish = d.pop("transactionalPublish")

        warming_provider = d.pop("warmingProvider")

        _channel_cost = d.pop("channelCost", UNSET)
        channel_cost: ApprovalCostRule | Unset
        if isinstance(_channel_cost,  Unset):
            channel_cost = UNSET
        else:
            channel_cost = ApprovalCostRule.from_dict(_channel_cost)




        _client_signoff = d.pop("clientSignoff", UNSET)
        client_signoff: ApprovalClientSignoff | Unset
        if isinstance(_client_signoff,  Unset):
            client_signoff = UNSET
        else:
            client_signoff = ApprovalClientSignoff.from_dict(_client_signoff)




        approval_policy = cls(
            approval_lists=approval_lists,
            approvals_required=approvals_required,
            audience_min=audience_min,
            author_lacks_send=author_lacks_send,
            automation_publish=automation_publish,
            emergency_bypass=emergency_bypass,
            new_provider_days=new_provider_days,
            post_publish=post_publish,
            ready_with_warnings=ready_with_warnings,
            transactional_publish=transactional_publish,
            warming_provider=warming_provider,
            channel_cost=channel_cost,
            client_signoff=client_signoff,
        )

        return approval_policy

