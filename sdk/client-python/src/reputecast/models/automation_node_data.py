from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_condition import AutomationCondition
  from ..models.automation_duration import AutomationDuration
  from ..models.automation_email_step import AutomationEmailStep
  from ..models.automation_list_action import AutomationListAction
  from ..models.automation_push_step import AutomationPushStep
  from ..models.automation_sms_step import AutomationSMSStep
  from ..models.automation_split_variant import AutomationSplitVariant
  from ..models.automation_trigger import AutomationTrigger
  from ..models.automation_webhook import AutomationWebhook
  from ..models.automation_whats_app_step import AutomationWhatsAppStep





T = TypeVar("T", bound="AutomationNodeData")



@_attrs_define
class AutomationNodeData:
    """ 
        Attributes:
            action (AutomationListAction | Unset):
            condition (AutomationCondition | Unset):
            delay (AutomationDuration | Unset):
            email (AutomationEmailStep | Unset):
            label (str | Unset):
            push (AutomationPushStep | Unset):
            sms (AutomationSMSStep | Unset):
            trigger (AutomationTrigger | Unset):
            variants (list[AutomationSplitVariant] | Unset):
            webhook (AutomationWebhook | Unset):
            whatsapp (AutomationWhatsAppStep | Unset):
     """

    action: AutomationListAction | Unset = UNSET
    condition: AutomationCondition | Unset = UNSET
    delay: AutomationDuration | Unset = UNSET
    email: AutomationEmailStep | Unset = UNSET
    label: str | Unset = UNSET
    push: AutomationPushStep | Unset = UNSET
    sms: AutomationSMSStep | Unset = UNSET
    trigger: AutomationTrigger | Unset = UNSET
    variants: list[AutomationSplitVariant] | Unset = UNSET
    webhook: AutomationWebhook | Unset = UNSET
    whatsapp: AutomationWhatsAppStep | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_condition import AutomationCondition # noqa: PLC0415
        from ..models.automation_duration import AutomationDuration # noqa: PLC0415
        from ..models.automation_email_step import AutomationEmailStep # noqa: PLC0415
        from ..models.automation_list_action import AutomationListAction # noqa: PLC0415
        from ..models.automation_push_step import AutomationPushStep # noqa: PLC0415
        from ..models.automation_sms_step import AutomationSMSStep # noqa: PLC0415
        from ..models.automation_split_variant import AutomationSplitVariant # noqa: PLC0415
        from ..models.automation_trigger import AutomationTrigger # noqa: PLC0415
        from ..models.automation_webhook import AutomationWebhook # noqa: PLC0415
        from ..models.automation_whats_app_step import AutomationWhatsAppStep # noqa: PLC0415
        action: dict[str, Any] | Unset = UNSET
        if not isinstance(self.action, Unset):
            action = self.action.to_dict()

        condition: dict[str, Any] | Unset = UNSET
        if not isinstance(self.condition, Unset):
            condition = self.condition.to_dict()

        delay: dict[str, Any] | Unset = UNSET
        if not isinstance(self.delay, Unset):
            delay = self.delay.to_dict()

        email: dict[str, Any] | Unset = UNSET
        if not isinstance(self.email, Unset):
            email = self.email.to_dict()

        label = self.label

        push: dict[str, Any] | Unset = UNSET
        if not isinstance(self.push, Unset):
            push = self.push.to_dict()

        sms: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sms, Unset):
            sms = self.sms.to_dict()

        trigger: dict[str, Any] | Unset = UNSET
        if not isinstance(self.trigger, Unset):
            trigger = self.trigger.to_dict()

        variants: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.variants, Unset):
            variants = []
            for variants_item_data in self.variants:
                variants_item = variants_item_data.to_dict()
                variants.append(variants_item)



        webhook: dict[str, Any] | Unset = UNSET
        if not isinstance(self.webhook, Unset):
            webhook = self.webhook.to_dict()

        whatsapp: dict[str, Any] | Unset = UNSET
        if not isinstance(self.whatsapp, Unset):
            whatsapp = self.whatsapp.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if action is not UNSET:
            field_dict["action"] = action
        if condition is not UNSET:
            field_dict["condition"] = condition
        if delay is not UNSET:
            field_dict["delay"] = delay
        if email is not UNSET:
            field_dict["email"] = email
        if label is not UNSET:
            field_dict["label"] = label
        if push is not UNSET:
            field_dict["push"] = push
        if sms is not UNSET:
            field_dict["sms"] = sms
        if trigger is not UNSET:
            field_dict["trigger"] = trigger
        if variants is not UNSET:
            field_dict["variants"] = variants
        if webhook is not UNSET:
            field_dict["webhook"] = webhook
        if whatsapp is not UNSET:
            field_dict["whatsapp"] = whatsapp

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_condition import AutomationCondition # noqa: PLC0415
        from ..models.automation_duration import AutomationDuration # noqa: PLC0415
        from ..models.automation_email_step import AutomationEmailStep # noqa: PLC0415
        from ..models.automation_list_action import AutomationListAction # noqa: PLC0415
        from ..models.automation_push_step import AutomationPushStep # noqa: PLC0415
        from ..models.automation_sms_step import AutomationSMSStep # noqa: PLC0415
        from ..models.automation_split_variant import AutomationSplitVariant # noqa: PLC0415
        from ..models.automation_trigger import AutomationTrigger # noqa: PLC0415
        from ..models.automation_webhook import AutomationWebhook # noqa: PLC0415
        from ..models.automation_whats_app_step import AutomationWhatsAppStep # noqa: PLC0415
        d = dict(src_dict)
        _action = d.pop("action", UNSET)
        action: AutomationListAction | Unset
        if isinstance(_action,  Unset):
            action = UNSET
        else:
            action = AutomationListAction.from_dict(_action)




        _condition = d.pop("condition", UNSET)
        condition: AutomationCondition | Unset
        if isinstance(_condition,  Unset):
            condition = UNSET
        else:
            condition = AutomationCondition.from_dict(_condition)




        _delay = d.pop("delay", UNSET)
        delay: AutomationDuration | Unset
        if isinstance(_delay,  Unset):
            delay = UNSET
        else:
            delay = AutomationDuration.from_dict(_delay)




        _email = d.pop("email", UNSET)
        email: AutomationEmailStep | Unset
        if isinstance(_email,  Unset):
            email = UNSET
        else:
            email = AutomationEmailStep.from_dict(_email)




        label = d.pop("label", UNSET)

        _push = d.pop("push", UNSET)
        push: AutomationPushStep | Unset
        if isinstance(_push,  Unset):
            push = UNSET
        else:
            push = AutomationPushStep.from_dict(_push)




        _sms = d.pop("sms", UNSET)
        sms: AutomationSMSStep | Unset
        if isinstance(_sms,  Unset):
            sms = UNSET
        else:
            sms = AutomationSMSStep.from_dict(_sms)




        _trigger = d.pop("trigger", UNSET)
        trigger: AutomationTrigger | Unset
        if isinstance(_trigger,  Unset):
            trigger = UNSET
        else:
            trigger = AutomationTrigger.from_dict(_trigger)




        _variants = d.pop("variants", UNSET)
        variants: list[AutomationSplitVariant] | Unset = UNSET
        if _variants is not UNSET:
            variants = []
            for variants_item_data in _variants:
                variants_item = AutomationSplitVariant.from_dict(variants_item_data)



                variants.append(variants_item)


        _webhook = d.pop("webhook", UNSET)
        webhook: AutomationWebhook | Unset
        if isinstance(_webhook,  Unset):
            webhook = UNSET
        else:
            webhook = AutomationWebhook.from_dict(_webhook)




        _whatsapp = d.pop("whatsapp", UNSET)
        whatsapp: AutomationWhatsAppStep | Unset
        if isinstance(_whatsapp,  Unset):
            whatsapp = UNSET
        else:
            whatsapp = AutomationWhatsAppStep.from_dict(_whatsapp)




        automation_node_data = cls(
            action=action,
            condition=condition,
            delay=delay,
            email=email,
            label=label,
            push=push,
            sms=sms,
            trigger=trigger,
            variants=variants,
            webhook=webhook,
            whatsapp=whatsapp,
        )

        return automation_node_data

