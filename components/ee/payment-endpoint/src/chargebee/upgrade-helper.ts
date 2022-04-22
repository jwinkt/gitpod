/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the Gitpod Enterprise Source Code License,
 * See License.enterprise.txt in the project root folder.
 */

import { injectable, inject } from "inversify";
import { ChargebeeProvider } from "./chargebee-provider";
import { Chargebee as chargebee } from './chargebee-types';
import { LogContext, log } from "@gitpod/gitpod-protocol/lib/util/logging";

@injectable()
export class UpgradeHelper {
    @inject(ChargebeeProvider) protected readonly chargebeeProvider: ChargebeeProvider;

    /**
     * Uses subscription.add_charge_at_term_end to 'manually' add a charge to the given Chargebee Subscription
     * (see https://apidocs.chargebee.com/docs/api/subscriptions#add_charge_at_term_end)
     *
     * @param userId
     * @param chargebeeSubscriptionId
     * @param amountInCents
     * @param description
     * @param upgradeTimestamp
     */
    async chargeForUpgrade(userId: string, chargebeeSubscriptionId: string, amountInCents: number, description: string, upgradeTimestamp: string) {
        const logContext: LogContext = { userId };
        const logPayload = { chargebeeSubscriptionId: chargebeeSubscriptionId, amountInCents, description, upgradeTimestamp };

        await new Promise<void>((resolve, reject) => {
            log.info(logContext, 'Charge on Upgrade: Upgrade detected.', logPayload);
            this.chargebeeProvider.subscription.add_charge_at_term_end(chargebeeSubscriptionId, {
                amount: amountInCents,
                description
            }).request(function (error: any, result: any) {
                if (error) {
                    log.error(logContext, 'Charge on Upgrade: error', error, logPayload);
                    reject(error);
                } else {
                    log.info(logContext, 'Charge on Upgrade: successful', logPayload);
                    resolve();
                }
            });
        });
    }

    // Returns a ratio between 0 and 1:
    //     0 means we've just finished the term
    //     1 means we still have the entire term left
    getCurrentTermRemainingRatio(chargebeeSubscription: chargebee.Subscription): number {
        if (!chargebeeSubscription.next_billing_at) {
            throw new Error('subscription.next_billing_at must be set.');
        }
        const now = new Date();
        const nextBilling = new Date(chargebeeSubscription.next_billing_at * 1000);
        const remainingMs = nextBilling.getTime() - now.getTime();

        const unitToBillingPeriodInDays = { day: 1, week: 7, month: 365.25 / 12, year: 365.25 };
        let billingPeriodInDays;
        if (typeof chargebeeSubscription.billing_period === "number" && !!chargebeeSubscription.billing_period_unit) {
            billingPeriodInDays = chargebeeSubscription.billing_period * unitToBillingPeriodInDays[chargebeeSubscription.billing_period_unit];
        } else {
            billingPeriodInDays = 1 * unitToBillingPeriodInDays["month"];
        }
        const billingPeriodMs = 1000 * 3600 * 24 * billingPeriodInDays;

        return remainingMs / billingPeriodMs;
    }
}